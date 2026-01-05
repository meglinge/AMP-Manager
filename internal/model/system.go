package model

import "time"

// RetryConfigResponse 重试配置响应
type RetryConfigResponse struct {
	Enabled           bool  `json:"enabled"`
	MaxAttempts       int   `json:"maxAttempts"`
	GateTimeoutMs     int64 `json:"gateTimeoutMs"`
	MaxBodyBytes      int64 `json:"maxBodyBytes"`
	BackoffBaseMs     int64 `json:"backoffBaseMs"`
	BackoffMaxMs      int64 `json:"backoffMaxMs"`
	RetryOn429        bool  `json:"retryOn429"`
	RetryOn5xx        bool  `json:"retryOn5xx"`
	RespectRetryAfter bool  `json:"respectRetryAfter"`
}

// RetryConfigRequest 重试配置请求
type RetryConfigRequest struct {
	Enabled           bool  `json:"enabled"`
	MaxAttempts       int   `json:"maxAttempts"`
	GateTimeoutMs     int64 `json:"gateTimeoutMs"`
	MaxBodyBytes      int64 `json:"maxBodyBytes"`
	BackoffBaseMs     int64 `json:"backoffBaseMs"`
	BackoffMaxMs      int64 `json:"backoffMaxMs"`
	RetryOn429        bool  `json:"retryOn429"`
	RetryOn5xx        bool  `json:"retryOn5xx"`
	RespectRetryAfter bool  `json:"respectRetryAfter"`
}

// SystemConfig 系统配置存储
type SystemConfig struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}
