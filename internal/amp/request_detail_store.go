package amp

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultDetailTTL      = 5 * time.Minute
	DetailCleanupInterval = 30 * time.Second
	MaxBodySize           = 1 * 1024 * 1024  // 1MB max body size to store
	MaxDetailEntries      = 10000            // 内存中最多保存的条目数
)

// RequestDetail stores request/response headers and bodies
type RequestDetail struct {
	RequestID              string
	CreatedAt              time.Time
	LastUpdatedAt          time.Time
	RequestHeaders         http.Header
	RequestBody            []byte
	TranslatedRequestBody  []byte // 翻译后发送给上游的请求体
	ResponseHeaders        http.Header
	ResponseBody           []byte
	TranslatedResponseBody []byte // 翻译后发送给客户端的响应体
	Persisted              bool
}

// RequestDetailStore stores request details in memory with TTL
type RequestDetailStore struct {
	mu       sync.RWMutex
	details  map[string]*RequestDetail
	db       *sql.DB
	ttl      time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
}

var (
	globalDetailStore *RequestDetailStore
	detailStoreOnce   sync.Once
	detailStoreMu     sync.Mutex
)

// InitRequestDetailStore initializes the global request detail store
func InitRequestDetailStore(db *sql.DB) {
	detailStoreOnce.Do(func() {
		globalDetailStore = NewRequestDetailStore(db, DefaultDetailTTL)
		log.Info("request detail store: initialized")
	})
}

// ReinitRequestDetailStore reinitializes the global request detail store (after db replacement)
func ReinitRequestDetailStore(db *sql.DB) {
	detailStoreMu.Lock()
	defer detailStoreMu.Unlock()
	if globalDetailStore != nil {
		globalDetailStore.Stop()
	}
	globalDetailStore = NewRequestDetailStore(db, DefaultDetailTTL)
	log.Info("request detail store: reinitialized")
}

// GetRequestDetailStore returns the global request detail store
func GetRequestDetailStore() *RequestDetailStore {
	return globalDetailStore
}

// StopRequestDetailStore stops the global request detail store
func StopRequestDetailStore() {
	if globalDetailStore != nil {
		globalDetailStore.Stop()
		log.Info("request detail store: stopped")
	}
}

// NewRequestDetailStore creates a new request detail store
func NewRequestDetailStore(db *sql.DB, ttl time.Duration) *RequestDetailStore {
	s := &RequestDetailStore{
		details:  make(map[string]*RequestDetail),
		db:       db,
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}
	s.wg.Add(1)
	go s.cleanupLoop()
	return s
}

// Store stores request detail in memory
func (s *RequestDetailStore) Store(detail *RequestDetail) {
	if detail == nil || detail.RequestID == "" {
		return
	}
	detail.CreatedAt = time.Now().UTC()
	detail.LastUpdatedAt = time.Now().UTC()

	s.mu.Lock()
	if len(s.details) >= MaxDetailEntries {
		s.evictOldestLocked()
	}
	s.details[detail.RequestID] = detail
	s.mu.Unlock()

	log.Debugf("request detail store: stored detail for %s", detail.RequestID)
}

// evictOldestLocked 驱逐最老的条目（必须在持锁状态下调用）
func (s *RequestDetailStore) evictOldestLocked() {
	var oldestID string
	var oldestTime time.Time
	first := true

	for id, detail := range s.details {
		if first || detail.LastUpdatedAt.Before(oldestTime) {
			oldestID = id
			oldestTime = detail.LastUpdatedAt
			first = false
		}
	}

	if oldestID != "" {
		delete(s.details, oldestID)
		log.Debugf("request detail store: evicted oldest entry %s due to max entries limit", oldestID)
	}
}

// UpdateRequestData updates the request headers and body
func (s *RequestDetailStore) UpdateRequestData(requestID string, headers http.Header, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	detail, exists := s.details[requestID]
	if !exists {
		now := time.Now().UTC()
		detail = &RequestDetail{
			RequestID:     requestID,
			CreatedAt:     now,
			LastUpdatedAt: now,
		}
		s.details[requestID] = detail
	}
	detail.LastUpdatedAt = time.Now().UTC()
	detail.RequestHeaders = headers.Clone()
	if len(body) <= MaxBodySize {
		detail.RequestBody = make([]byte, len(body))
		copy(detail.RequestBody, body)
	} else {
		detail.RequestBody = make([]byte, MaxBodySize)
		copy(detail.RequestBody, body[:MaxBodySize])
	}
}

// UpdateTranslatedRequestBody stores the translated request body
func (s *RequestDetailStore) UpdateTranslatedRequestBody(requestID string, body []byte) {
	if len(body) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	detail, exists := s.details[requestID]
	if !exists {
		now := time.Now().UTC()
		detail = &RequestDetail{
			RequestID:     requestID,
			CreatedAt:     now,
			LastUpdatedAt: now,
		}
		s.details[requestID] = detail
	}
	detail.LastUpdatedAt = time.Now().UTC()
	if len(body) <= MaxBodySize {
		detail.TranslatedRequestBody = make([]byte, len(body))
		copy(detail.TranslatedRequestBody, body)
	} else {
		detail.TranslatedRequestBody = make([]byte, MaxBodySize)
		copy(detail.TranslatedRequestBody, body[:MaxBodySize])
	}
}

// UpdateResponseData updates the response headers and body
func (s *RequestDetailStore) UpdateResponseData(requestID string, headers http.Header, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	detail, exists := s.details[requestID]
	if !exists {
		now := time.Now().UTC()
		detail = &RequestDetail{
			RequestID:     requestID,
			CreatedAt:     now,
			LastUpdatedAt: now,
		}
		s.details[requestID] = detail
	}
	detail.LastUpdatedAt = time.Now().UTC()
	detail.ResponseHeaders = headers.Clone()
	if len(body) <= MaxBodySize {
		detail.ResponseBody = make([]byte, len(body))
		copy(detail.ResponseBody, body)
	} else {
		detail.ResponseBody = make([]byte, MaxBodySize)
		copy(detail.ResponseBody, body[:MaxBodySize])
	}
}

// AppendTranslatedResponse appends translated response data for debugging
func (s *RequestDetailStore) AppendTranslatedResponse(requestID string, data []byte) {
	if len(data) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	detail, exists := s.details[requestID]
	if !exists {
		return
	}

	// 限制翻译响应大小
	currentLen := len(detail.TranslatedResponseBody)
	if currentLen >= MaxBodySize {
		return
	}

	remaining := MaxBodySize - currentLen
	if len(data) > remaining {
		data = data[:remaining]
	}

	detail.TranslatedResponseBody = append(detail.TranslatedResponseBody, data...)
	detail.LastUpdatedAt = time.Now().UTC()
}

// copyDetail creates a deep copy of a RequestDetail
func copyDetail(detail *RequestDetail) *RequestDetail {
	copied := &RequestDetail{
		RequestID:              detail.RequestID,
		CreatedAt:              detail.CreatedAt,
		LastUpdatedAt:          detail.LastUpdatedAt,
		RequestHeaders:         nil,
		RequestBody:            make([]byte, len(detail.RequestBody)),
		TranslatedRequestBody:  make([]byte, len(detail.TranslatedRequestBody)),
		ResponseHeaders:        nil,
		ResponseBody:           make([]byte, len(detail.ResponseBody)),
		TranslatedResponseBody: make([]byte, len(detail.TranslatedResponseBody)),
		Persisted:              detail.Persisted,
	}
	copy(copied.RequestBody, detail.RequestBody)
	copy(copied.TranslatedRequestBody, detail.TranslatedRequestBody)
	copy(copied.ResponseBody, detail.ResponseBody)
	copy(copied.TranslatedResponseBody, detail.TranslatedResponseBody)
	if detail.RequestHeaders != nil {
		copied.RequestHeaders = detail.RequestHeaders.Clone()
	}
	if detail.ResponseHeaders != nil {
		copied.ResponseHeaders = detail.ResponseHeaders.Clone()
	}
	return copied
}

// Get retrieves request detail by ID (from memory first, then database)
func (s *RequestDetailStore) Get(requestID string) *RequestDetail {
	s.mu.RLock()
	detail, exists := s.details[requestID]
	if exists {
		copied := copyDetail(detail)
		s.mu.RUnlock()
		return copied
	}
	s.mu.RUnlock()

	return s.getFromDB(requestID)
}

// getFromDB retrieves request detail from database
func (s *RequestDetailStore) getFromDB(requestID string) *RequestDetail {
	if s.db == nil {
		return nil
	}

	var detail RequestDetail
	var requestHeaders, requestBody, responseHeaders, responseBody sql.NullString

	err := s.db.QueryRow(`
		SELECT request_id, request_headers, request_body, response_headers, response_body, created_at
		FROM request_log_details
		WHERE request_id = ?
	`, requestID).Scan(
		&detail.RequestID,
		&requestHeaders,
		&requestBody,
		&responseHeaders,
		&responseBody,
		&detail.CreatedAt,
	)

	if err != nil {
		if err != sql.ErrNoRows {
			log.Errorf("request detail store: failed to query from db: %v", err)
		}
		return nil
	}

	if requestHeaders.Valid {
		detail.RequestHeaders = parseHeadersJSON(requestHeaders.String)
	}
	if responseHeaders.Valid {
		detail.ResponseHeaders = parseHeadersJSON(responseHeaders.String)
	}
	if requestBody.Valid {
		detail.RequestBody = []byte(requestBody.String)
	}
	if responseBody.Valid {
		detail.ResponseBody = []byte(responseBody.String)
	}
	detail.LastUpdatedAt = detail.CreatedAt
	detail.Persisted = true

	return &detail
}

// persistToDB persists request detail to database
func (s *RequestDetailStore) persistToDB(detail *RequestDetail) error {
	if s.db == nil || detail == nil {
		return nil
	}

	requestHeadersJSON := headersToJSON(detail.RequestHeaders)
	responseHeadersJSON := headersToJSON(detail.ResponseHeaders)

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO request_log_details 
		(request_id, request_headers, request_body, response_headers, response_body, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		detail.RequestID,
		requestHeadersJSON,
		string(detail.RequestBody),
		responseHeadersJSON,
		string(detail.ResponseBody),
		detail.CreatedAt.Format(time.RFC3339),
	)

	if err != nil {
		log.Errorf("request detail store: failed to persist to db: %v", err)
		return err
	}

	log.Debugf("request detail store: persisted detail for %s", detail.RequestID)
	return nil
}

// cleanupLoop periodically cleans up expired entries and persists them to database
func (s *RequestDetailStore) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(DetailCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.stopChan:
			s.persistAll()
			return
		}
	}
}

// cleanup removes expired entries and persists them to database
func (s *RequestDetailStore) cleanup() {
	now := time.Now().UTC()
	var expiredIDs []string
	var snapshots []*RequestDetail

	s.mu.RLock()
	for id, detail := range s.details {
		if now.Sub(detail.LastUpdatedAt) > s.ttl {
			expiredIDs = append(expiredIDs, id)
			if !detail.Persisted {
				snapshots = append(snapshots, copyDetail(detail))
			}
		}
	}
	s.mu.RUnlock()

	persistedIDs := make(map[string]bool)
	for _, snapshot := range snapshots {
		if err := s.persistToDB(snapshot); err == nil {
			persistedIDs[snapshot.RequestID] = true
		}
	}

	if len(expiredIDs) > 0 {
		s.mu.Lock()
		for _, id := range expiredIDs {
			detail, exists := s.details[id]
			if exists && (detail.Persisted || persistedIDs[id]) {
				delete(s.details, id)
			}
		}
		s.mu.Unlock()
	}

	if len(persistedIDs) > 0 {
		log.Debugf("request detail store: cleaned up %d expired entries", len(persistedIDs))
	}
}

// persistAll persists all entries to database (called on shutdown)
func (s *RequestDetailStore) persistAll() {
	s.mu.RLock()
	snapshots := make([]*RequestDetail, 0, len(s.details))
	for _, detail := range s.details {
		if !detail.Persisted {
			snapshots = append(snapshots, copyDetail(detail))
		}
	}
	s.mu.RUnlock()

	count := 0
	for _, snapshot := range snapshots {
		if err := s.persistToDB(snapshot); err == nil {
			count++
		}
	}

	if count > 0 {
		log.Infof("request detail store: persisted %d entries on shutdown", count)
	}
}

// Stop stops the cleanup loop
func (s *RequestDetailStore) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// Helper functions for JSON serialization of headers
func headersToJSON(headers http.Header) string {
	if headers == nil {
		return "{}"
	}
	data, err := json.Marshal(headers)
	if err != nil {
		log.Warnf("request detail store: failed to marshal headers: %v", err)
		return "{}"
	}
	return string(data)
}

func parseHeadersJSON(jsonStr string) http.Header {
	headers := make(http.Header)
	if jsonStr == "" || jsonStr == "{}" {
		return headers
	}

	var data map[string][]string
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		log.Warnf("request detail store: failed to parse headers JSON: %v", err)
		return headers
	}

	for k, v := range data {
		headers[k] = v
	}
	return headers
}
