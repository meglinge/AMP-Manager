package service

import (
	"ampmanager/internal/repository"
)

const (
	retryConfigKey          = "retry_config"
	requestDetailEnabledKey = "request_detail_enabled"
	timeoutConfigKey        = "timeout_config"
	cacheTTLOverrideKey     = "cache_ttl_override"
)

type SystemConfigService struct {
	repo *repository.SystemConfigRepository
}

func NewSystemConfigService() *SystemConfigService {
	return &SystemConfigService{
		repo: repository.NewSystemConfigRepository(),
	}
}

// GetRetryConfigJSON 获取重试配置的 JSON 字符串
func (s *SystemConfigService) GetRetryConfigJSON() (string, error) {
	return s.repo.Get(retryConfigKey)
}

// SetRetryConfigJSON 保存重试配置的 JSON 字符串
func (s *SystemConfigService) SetRetryConfigJSON(value string) error {
	return s.repo.Set(retryConfigKey, value)
}

// GetRequestDetailEnabled 获取请求详情监控是否启用
func (s *SystemConfigService) GetRequestDetailEnabled() (bool, error) {
	value, err := s.repo.Get(requestDetailEnabledKey)
	if err != nil {
		return true, nil // 默认启用
	}
	return value != "false", nil
}

// SetRequestDetailEnabled 设置请求详情监控是否启用
func (s *SystemConfigService) SetRequestDetailEnabled(enabled bool) error {
	value := "true"
	if !enabled {
		value = "false"
	}
	return s.repo.Set(requestDetailEnabledKey, value)
}

// GetTimeoutConfigJSON 获取超时配置的 JSON 字符串
func (s *SystemConfigService) GetTimeoutConfigJSON() (string, error) {
	return s.repo.Get(timeoutConfigKey)
}

// GetCacheTTLOverride 获取缓存 TTL 覆盖配置
func (s *SystemConfigService) GetCacheTTLOverride() (string, error) {
	return s.repo.Get(cacheTTLOverrideKey)
}
