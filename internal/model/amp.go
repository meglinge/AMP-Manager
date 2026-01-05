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
	ProxyBaseURL  string `json:"proxyBaseUrl"`
	ConfigExample string `json:"configExample"`
	HasSettings   bool   `json:"hasSettings"`
	HasAPIKey     bool   `json:"hasApiKey"`
}
