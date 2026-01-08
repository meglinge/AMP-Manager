package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ampmanager/internal/model"
	"ampmanager/internal/repository"
)

var (
	ErrChannelNotFound = errors.New("渠道不存在")
)

type ChannelService struct {
	repo      *repository.ChannelRepository
	rrCounter sync.Map // map[string]*atomic.Uint64
}

func NewChannelService() *ChannelService {
	return &ChannelService{
		repo: repository.NewChannelRepository(),
	}
}

// getRRCounter 获取或创建指定 key 的原子计数器
func (s *ChannelService) getRRCounter(key string) *atomic.Uint64 {
	if v, ok := s.rrCounter.Load(key); ok {
		return v.(*atomic.Uint64)
	}
	counter := &atomic.Uint64{}
	actual, _ := s.rrCounter.LoadOrStore(key, counter)
	return actual.(*atomic.Uint64)
}

func (s *ChannelService) Create(req *model.ChannelRequest) (*model.ChannelResponse, error) {
	modelsJSON, _ := json.Marshal(req.Models)
	if req.Models == nil {
		modelsJSON = []byte("[]")
	}
	headersJSON, _ := json.Marshal(req.Headers)
	if req.Headers == nil {
		headersJSON = []byte("{}")
	}

	weight := req.Weight
	if weight < 1 {
		weight = 1
	}
	priority := req.Priority
	if priority < 1 {
		priority = 100
	}

	endpoint := req.Endpoint
	if endpoint == "" {
		endpoint = s.defaultEndpointForType(req.Type)
	}

	channel := &model.Channel{
		Type:        req.Type,
		Endpoint:    endpoint,
		Name:        req.Name,
		BaseURL:     strings.TrimSuffix(req.BaseURL, "/"),
		APIKey:      req.APIKey,
		Enabled:     req.Enabled,
		Weight:      weight,
		Priority:    priority,
		ModelsJSON:  string(modelsJSON),
		HeadersJSON: string(headersJSON),
	}

	if err := s.repo.Create(channel); err != nil {
		return nil, err
	}

	return s.toResponse(channel), nil
}

func (s *ChannelService) defaultEndpointForType(channelType model.ChannelType) model.ChannelEndpoint {
	switch channelType {
	case model.ChannelTypeOpenAI:
		return model.ChannelEndpointChatCompletions
	case model.ChannelTypeClaude:
		return model.ChannelEndpointMessages
	case model.ChannelTypeGemini:
		return model.ChannelEndpointGenerateContent
	default:
		return model.ChannelEndpointChatCompletions
	}
}

func (s *ChannelService) GetByID(id string) (*model.ChannelResponse, error) {
	channel, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}
	return s.toResponse(channel), nil
}

func (s *ChannelService) List() ([]*model.ChannelResponse, error) {
	channels, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	responses := make([]*model.ChannelResponse, len(channels))
	for i, ch := range channels {
		responses[i] = s.toResponse(ch)
	}
	return responses, nil
}

func (s *ChannelService) Update(id string, req *model.ChannelRequest) (*model.ChannelResponse, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrChannelNotFound
	}

	modelsJSON, _ := json.Marshal(req.Models)
	if req.Models == nil {
		modelsJSON = []byte("[]")
	}
	headersJSON, _ := json.Marshal(req.Headers)
	if req.Headers == nil {
		headersJSON = []byte("{}")
	}

	weight := req.Weight
	if weight < 1 {
		weight = 1
	}
	priority := req.Priority
	if priority < 1 {
		priority = 100
	}

	endpoint := req.Endpoint
	if endpoint == "" {
		endpoint = s.defaultEndpointForType(req.Type)
	}

	existing.Type = req.Type
	existing.Endpoint = endpoint
	existing.Name = req.Name
	existing.BaseURL = strings.TrimSuffix(req.BaseURL, "/")
	existing.Enabled = req.Enabled
	existing.Weight = weight
	existing.Priority = priority
	existing.ModelsJSON = string(modelsJSON)
	existing.HeadersJSON = string(headersJSON)

	if req.APIKey != "" {
		existing.APIKey = req.APIKey
	}

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}

	return s.toResponse(existing), nil
}

func (s *ChannelService) Delete(id string) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrChannelNotFound
	}
	return s.repo.Delete(id)
}

func (s *ChannelService) SetEnabled(id string, enabled bool) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrChannelNotFound
	}
	return s.repo.SetEnabled(id, enabled)
}

func (s *ChannelService) TestConnection(id string) (*model.TestChannelResponse, error) {
	channel, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var testURL string

	switch channel.Type {
	case model.ChannelTypeOpenAI:
		testURL = channel.BaseURL + "/v1/models"
	case model.ChannelTypeClaude:
		testURL = channel.BaseURL + "/v1/models"
	case model.ChannelTypeGemini:
		testURL = channel.BaseURL + "/v1beta/models"
	default:
		testURL = channel.BaseURL
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return &model.TestChannelResponse{
			Success: false,
			Message: fmt.Sprintf("创建请求失败: %v", err),
		}, nil
	}

	switch channel.Type {
	case model.ChannelTypeOpenAI:
		req.Header.Set("Authorization", "Bearer "+channel.APIKey)
	case model.ChannelTypeClaude:
		req.Header.Set("x-api-key", channel.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case model.ChannelTypeGemini:
		q := req.URL.Query()
		q.Set("key", channel.APIKey)
		req.URL.RawQuery = q.Encode()
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &model.TestChannelResponse{
			Success:   false,
			Message:   fmt.Sprintf("连接失败: %v", err),
			LatencyMs: latency,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &model.TestChannelResponse{
			Success:   true,
			Message:   fmt.Sprintf("连接成功 (HTTP %d)", resp.StatusCode),
			LatencyMs: latency,
		}, nil
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return &model.TestChannelResponse{
			Success:   false,
			Message:   fmt.Sprintf("认证失败 (HTTP %d)", resp.StatusCode),
			LatencyMs: latency,
		}, nil
	}

	return &model.TestChannelResponse{
		Success:   false,
		Message:   fmt.Sprintf("请求失败: HTTP %d", resp.StatusCode),
		LatencyMs: latency,
	}, nil
}

func (s *ChannelService) SelectChannelForModel(modelName string) (*model.Channel, error) {
	channels, err := s.repo.ListEnabled()
	if err != nil {
		return nil, err
	}

	var candidates []*model.Channel
	for _, ch := range channels {
		if s.channelMatchesModel(ch, modelName) {
			candidates = append(candidates, ch)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	minPriority := candidates[0].Priority
	for _, c := range candidates {
		if c.Priority < minPriority {
			minPriority = c.Priority
		}
	}

	var priorityCandidates []*model.Channel
	for _, c := range candidates {
		if c.Priority == minPriority {
			priorityCandidates = append(priorityCandidates, c)
		}
	}

	// 按 ID 排序确保稳定顺序
	sort.Slice(priorityCandidates, func(i, j int) bool {
		return priorityCandidates[i].ID < priorityCandidates[j].ID
	})

	// 使用原子计数器实现线程安全的 round-robin
	counter := s.getRRCounter(modelName)
	idx := int(counter.Add(1) - 1)
	selected := priorityCandidates[idx%len(priorityCandidates)]

	return selected, nil
}

func (s *ChannelService) channelMatchesModel(channel *model.Channel, modelName string) bool {
	var models []model.ChannelModel
	if err := json.Unmarshal([]byte(channel.ModelsJSON), &models); err != nil {
		return false
	}

	if len(models) == 0 {
		return s.defaultModelMatch(channel.Type, modelName)
	}

	modelLower := strings.ToLower(modelName)
	for _, m := range models {
		if strings.EqualFold(m.Name, modelName) || strings.EqualFold(m.Alias, modelName) {
			return true
		}
		nameLower := strings.ToLower(m.Name)
		if strings.Contains(nameLower, "*") {
			if s.wildcardMatch(nameLower, modelLower) {
				return true
			}
		}
	}

	return false
}

func (s *ChannelService) defaultModelMatch(channelType model.ChannelType, modelName string) bool {
	modelLower := strings.ToLower(modelName)
	switch channelType {
	case model.ChannelTypeGemini:
		return strings.HasPrefix(modelLower, "gemini")
	case model.ChannelTypeClaude:
		return strings.HasPrefix(modelLower, "claude")
	case model.ChannelTypeOpenAI:
		return strings.HasPrefix(modelLower, "gpt") || strings.HasPrefix(modelLower, "o1") || strings.HasPrefix(modelLower, "o3") || strings.HasPrefix(modelLower, "o4")
	}
	return false
}

func (s *ChannelService) wildcardMatch(pattern, text string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(text, strings.Trim(pattern, "*"))
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(text, strings.TrimPrefix(pattern, "*"))
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(text, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == text
}

func (s *ChannelService) GetChannelInternal(id string) (*model.Channel, error) {
	return s.repo.GetByID(id)
}

func (s *ChannelService) toResponse(channel *model.Channel) *model.ChannelResponse {
	var models []model.ChannelModel
	_ = json.Unmarshal([]byte(channel.ModelsJSON), &models)
	if models == nil {
		models = []model.ChannelModel{}
	}

	var headers map[string]string
	_ = json.Unmarshal([]byte(channel.HeadersJSON), &headers)
	if headers == nil {
		headers = map[string]string{}
	}

	return &model.ChannelResponse{
		ID:        channel.ID,
		Type:      channel.Type,
		Endpoint:  channel.Endpoint,
		Name:      channel.Name,
		BaseURL:   channel.BaseURL,
		APIKeySet: channel.APIKey != "",
		Enabled:   channel.Enabled,
		Weight:    channel.Weight,
		Priority:  channel.Priority,
		Models:    models,
		Headers:   headers,
		CreatedAt: channel.CreatedAt,
		UpdatedAt: channel.UpdatedAt,
	}
}
