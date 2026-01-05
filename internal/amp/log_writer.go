package amp

import (
	"database/sql"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// LogEntry 日志条目，用于写入数据库
type LogEntry struct {
	ID                       string
	CreatedAt                time.Time
	UserID                   string
	APIKeyID                 string
	OriginalModel            *string
	MappedModel              *string
	Provider                 *string
	ChannelID                *string
	Endpoint                 *string
	Method                   string
	Path                     string
	StatusCode               int
	LatencyMs                int64
	IsStreaming              bool
	InputTokens              *int
	OutputTokens             *int
	CacheReadInputTokens     *int
	CacheCreationInputTokens *int
	ErrorType                *string
	RequestID                *string
}

// LogWriter 异步批量日志写入器
type LogWriter struct {
	db          *sql.DB
	entryChan   chan LogEntry
	batchSize   int
	flushInterval time.Duration
	wg          sync.WaitGroup
	stopChan    chan struct{}
	stopped     bool
	mu          sync.Mutex
}

// NewLogWriter 创建日志写入器
func NewLogWriter(db *sql.DB, bufferSize, batchSize int, flushInterval time.Duration) *LogWriter {
	w := &LogWriter{
		db:            db,
		entryChan:     make(chan LogEntry, bufferSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		stopChan:      make(chan struct{}),
	}
	w.wg.Add(1)
	go w.run()
	return w
}

// Write 异步写入日志（非阻塞）
func (w *LogWriter) Write(entry LogEntry) bool {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return false
	}
	w.mu.Unlock()

	select {
	case w.entryChan <- entry:
		return true
	default:
		log.Warn("log writer: queue full, dropping entry")
		return false
	}
}

// WriteFromTrace 从 RequestTrace 创建并写入日志
func (w *LogWriter) WriteFromTrace(trace *RequestTrace) bool {
	if trace == nil {
		return false
	}

	snapshot := trace.Clone()
	entry := LogEntry{
		ID:                       uuid.New().String(),
		CreatedAt:                snapshot.StartTime,
		UserID:                   snapshot.UserID,
		APIKeyID:                 snapshot.APIKeyID,
		Method:                   snapshot.Method,
		Path:                     snapshot.Path,
		StatusCode:               snapshot.StatusCode,
		LatencyMs:                snapshot.LatencyMs,
		IsStreaming:              snapshot.IsStreaming,
		InputTokens:              snapshot.InputTokens,
		OutputTokens:             snapshot.OutputTokens,
		CacheReadInputTokens:     snapshot.CacheReadInputTokens,
		CacheCreationInputTokens: snapshot.CacheCreationInputTokens,
	}

	if snapshot.OriginalModel != "" {
		entry.OriginalModel = &snapshot.OriginalModel
	}
	if snapshot.MappedModel != "" {
		entry.MappedModel = &snapshot.MappedModel
	}
	if snapshot.Provider != "" {
		entry.Provider = &snapshot.Provider
	}
	if snapshot.ChannelID != "" {
		entry.ChannelID = &snapshot.ChannelID
	}
	if snapshot.Endpoint != "" {
		entry.Endpoint = &snapshot.Endpoint
	}
	if snapshot.ErrorType != "" {
		entry.ErrorType = &snapshot.ErrorType
	}
	if snapshot.RequestID != "" {
		entry.RequestID = &snapshot.RequestID
	}

	return w.Write(entry)
}

// Stop 停止写入器并刷新剩余日志
func (w *LogWriter) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.mu.Unlock()

	close(w.stopChan)
	w.wg.Wait()
}

// run 后台运行的写入循环
func (w *LogWriter) run() {
	defer w.wg.Done()

	batch := make([]LogEntry, 0, w.batchSize)
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-w.entryChan:
			batch = append(batch, entry)
			if len(batch) >= w.batchSize {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				w.flush(batch)
				batch = batch[:0]
			}
		case <-w.stopChan:
			// 处理剩余的日志
			close(w.entryChan)
			for entry := range w.entryChan {
				batch = append(batch, entry)
			}
			if len(batch) > 0 {
				w.flush(batch)
			}
			return
		}
	}
}

// flush 批量写入数据库
func (w *LogWriter) flush(entries []LogEntry) {
	if len(entries) == 0 {
		return
	}

	tx, err := w.db.Begin()
	if err != nil {
		log.Errorf("log writer: failed to begin transaction: %v", err)
		return
	}

	stmt, err := tx.Prepare(`
		INSERT INTO request_logs (
			id, created_at, user_id, api_key_id, original_model, mapped_model,
			provider, channel_id, endpoint, method, path, status_code, latency_ms,
			is_streaming, input_tokens, output_tokens, cache_read_input_tokens,
			cache_creation_input_tokens, error_type, request_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Errorf("log writer: failed to prepare statement: %v", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	for _, e := range entries {
		isStreaming := 0
		if e.IsStreaming {
			isStreaming = 1
		}

		_, err := stmt.Exec(
			e.ID, e.CreatedAt, e.UserID, e.APIKeyID, e.OriginalModel, e.MappedModel,
			e.Provider, e.ChannelID, e.Endpoint, e.Method, e.Path, e.StatusCode, e.LatencyMs,
			isStreaming, e.InputTokens, e.OutputTokens, e.CacheReadInputTokens,
			e.CacheCreationInputTokens, e.ErrorType, e.RequestID,
		)
		if err != nil {
			log.Errorf("log writer: failed to insert entry: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("log writer: failed to commit transaction: %v", err)
		tx.Rollback()
		return
	}

	log.Debugf("log writer: flushed %d entries", len(entries))
}

// 全局日志写入器实例
var (
	globalLogWriter *LogWriter
	logWriterOnce   sync.Once
)

// InitLogWriter 初始化全局日志写入器
func InitLogWriter(db *sql.DB) {
	logWriterOnce.Do(func() {
		globalLogWriter = NewLogWriter(db, 10000, 100, 200*time.Millisecond)
		log.Info("log writer: initialized")
	})
}

// GetLogWriter 获取全局日志写入器
func GetLogWriter() *LogWriter {
	return globalLogWriter
}

// StopLogWriter 停止全局日志写入器
func StopLogWriter() {
	if globalLogWriter != nil {
		globalLogWriter.Stop()
		log.Info("log writer: stopped")
	}
}

// LoggingBodyWrapper 包装响应体，在 Close 时写入日志
type LoggingBodyWrapper struct {
	io.ReadCloser
	trace      *RequestTrace
	statusCode int
	once       sync.Once
}

// NewLoggingBodyWrapper 创建日志包装器
func NewLoggingBodyWrapper(body io.ReadCloser, trace *RequestTrace, statusCode int) *LoggingBodyWrapper {
	return &LoggingBodyWrapper{
		ReadCloser: body,
		trace:      trace,
		statusCode: statusCode,
	}
}

// Close 关闭并写入日志
func (w *LoggingBodyWrapper) Close() error {
	err := w.ReadCloser.Close()
	w.once.Do(func() {
		if w.trace != nil {
			w.trace.SetResponse(w.statusCode)
			if writer := GetLogWriter(); writer != nil {
				writer.WriteFromTrace(w.trace)
			}
		}
	})
	return err
}
