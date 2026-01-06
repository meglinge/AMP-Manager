package model

import "time"

type AmpSettings struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	UpstreamURL        string    `json:"upstream_url"`
	UpstreamAPIKey     string    `json:"-"`
	ModelMappingsJSON  string    `json:"-"`
	ForceModelMappings bool      `json:"force_model_mappings"`
	Enabled            bool      `json:"enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ModelMapping struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Regex         bool   `json:"regex"`
	ThinkingLevel string `json:"thinkingLevel,omitempty"`
}

type UserAPIKey struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	Prefix     string     `json:"prefix"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	LastUsed   *time.Time `json:"last_used,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Request/Response 结构体

type AmpSettingsRequest struct {
	UpstreamURL        string         `json:"upstreamUrl"`
	UpstreamAPIKey     string         `json:"upstreamApiKey,omitempty"`
	ModelMappings      []ModelMapping `json:"modelMappings,omitempty"`
	ForceModelMappings bool           `json:"forceModelMappings"`
	Enabled            bool           `json:"enabled"`
}

type AmpSettingsResponse struct {
	UpstreamURL        string         `json:"upstreamUrl"`
	ModelMappings      []ModelMapping `json:"modelMappings"`
	ForceModelMappings bool           `json:"forceModelMappings"`
	Enabled            bool           `json:"enabled"`
	HasAPIKey          bool           `json:"apiKeySet"`
	CreatedAt          time.Time      `json:"createdAt,omitempty"`
	UpdatedAt          time.Time      `json:"updatedAt,omitempty"`
}

type TestConnectionResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latencyMs,omitempty"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
}

type CreateAPIKeyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	APIKey    string    `json:"apiKey"`
	CreatedAt time.Time `json:"createdAt"`
	Message   string    `json:"message"`
}

type APIKeyListItem struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Prefix    string     `json:"prefix"`
	CreatedAt time.Time  `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	LastUsed  *time.Time `json:"lastUsedAt,omitempty"`
	IsActive  bool       `json:"isActive"`
}

type BootstrapResponse struct {
	HasSettings bool `json:"hasSettings"`
	HasAPIKey   bool `json:"hasApiKey"`
}

// RequestLogStatus 请求日志状态
type RequestLogStatus string

const (
	RequestLogStatusPending RequestLogStatus = "pending"
	RequestLogStatusSuccess RequestLogStatus = "success"
	RequestLogStatusError   RequestLogStatus = "error"
)

// RequestLog 请求日志记录
type RequestLog struct {
	ID                       string           `json:"id"`
	CreatedAt                string           `json:"createdAt"`
	UpdatedAt                *string          `json:"updatedAt,omitempty"`
	Status                   RequestLogStatus `json:"status"`
	UserID                   string           `json:"userId"`
	APIKeyID                 string           `json:"apiKeyId"`
	OriginalModel            *string          `json:"originalModel,omitempty"`
	MappedModel              *string          `json:"mappedModel,omitempty"`
	Provider                 *string          `json:"provider,omitempty"`
	ChannelID                *string          `json:"channelId,omitempty"`
	Endpoint                 *string          `json:"endpoint,omitempty"`
	Method                   string           `json:"method"`
	Path                     string           `json:"path"`
	StatusCode               int              `json:"statusCode"`
	LatencyMs                int64            `json:"latencyMs"`
	IsStreaming              bool             `json:"isStreaming"`
	InputTokens              *int             `json:"inputTokens,omitempty"`
	OutputTokens             *int             `json:"outputTokens,omitempty"`
	CacheReadInputTokens     *int             `json:"cacheReadInputTokens,omitempty"`
	CacheCreationInputTokens *int             `json:"cacheCreationInputTokens,omitempty"`
	ErrorType                *string          `json:"errorType,omitempty"`
	RequestID                *string          `json:"requestId,omitempty"`
}

// RequestLogListResponse 请求日志列表响应
type RequestLogListResponse struct {
	Items    []RequestLog `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// UsageSummary 用量统计
type UsageSummary struct {
	GroupKey                 string `json:"groupKey"`
	InputTokensSum           int64  `json:"inputTokensSum"`
	OutputTokensSum          int64  `json:"outputTokensSum"`
	CacheReadInputTokensSum  int64  `json:"cacheReadInputTokensSum"`
	CacheCreationInputTokensSum int64 `json:"cacheCreationInputTokensSum"`
	RequestCount             int64  `json:"requestCount"`
	ErrorCount               int64  `json:"errorCount"`
}

// UsageSummaryResponse 用量统计响应
type UsageSummaryResponse struct {
	Items []UsageSummary `json:"items"`
}
