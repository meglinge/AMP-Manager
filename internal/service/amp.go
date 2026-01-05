package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"ampmanager/internal/config"
	"ampmanager/internal/model"
	"ampmanager/internal/repository"
)

var (
	ErrAPIKeyNotFound = errors.New("API Key 不存在")
	ErrAPIKeyRevoked  = errors.New("API Key 已被撤销")
	ErrNotOwner       = errors.New("无权操作此资源")
)

type AmpService struct {
	settingsRepo *repository.AmpSettingsRepository
	apiKeyRepo   *repository.APIKeyRepository
}

func NewAmpService() *AmpService {
	return &AmpService{
		settingsRepo: repository.NewAmpSettingsRepository(),
		apiKeyRepo:   repository.NewAPIKeyRepository(),
	}
}

func (s *AmpService) GetSettings(userID string) (*model.AmpSettingsResponse, error) {
	settings, err := s.settingsRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return &model.AmpSettingsResponse{
			UpstreamURL:        "https://ampcode.com",
			ModelMappings:      []model.ModelMapping{},
			ForceModelMappings: false,
			Enabled:            false,
			HasAPIKey:          false,
		}, nil
	}

	var mappings []model.ModelMapping
	if settings.ModelMappingsJSON != "" {
		_ = json.Unmarshal([]byte(settings.ModelMappingsJSON), &mappings)
	}
	if mappings == nil {
		mappings = []model.ModelMapping{}
	}

	return &model.AmpSettingsResponse{
		UpstreamURL:        settings.UpstreamURL,
		ModelMappings:      mappings,
		ForceModelMappings: settings.ForceModelMappings,
		Enabled:            settings.Enabled,
		HasAPIKey:          settings.UpstreamAPIKey != "",
		CreatedAt:          settings.CreatedAt,
		UpdatedAt:          settings.UpdatedAt,
	}, nil
}

func (s *AmpService) UpdateSettings(userID string, req *model.AmpSettingsRequest) (*model.AmpSettingsResponse, error) {
	existing, err := s.settingsRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	settings := &model.AmpSettings{
		UserID:             userID,
		UpstreamURL:        req.UpstreamURL,
		ForceModelMappings: req.ForceModelMappings,
		Enabled:            req.Enabled,
	}

	if existing != nil && req.UpstreamAPIKey == "" {
		settings.UpstreamAPIKey = existing.UpstreamAPIKey
	} else {
		settings.UpstreamAPIKey = req.UpstreamAPIKey
	}

	if req.ModelMappings != nil {
		mappingsJSON, _ := json.Marshal(req.ModelMappings)
		settings.ModelMappingsJSON = string(mappingsJSON)
	} else if existing != nil {
		settings.ModelMappingsJSON = existing.ModelMappingsJSON
	}

	if err := s.settingsRepo.Upsert(settings); err != nil {
		return nil, err
	}

	var mappings []model.ModelMapping
	if settings.ModelMappingsJSON != "" {
		_ = json.Unmarshal([]byte(settings.ModelMappingsJSON), &mappings)
	}
	if mappings == nil {
		mappings = []model.ModelMapping{}
	}

	return &model.AmpSettingsResponse{
		UpstreamURL:        settings.UpstreamURL,
		ModelMappings:      mappings,
		ForceModelMappings: settings.ForceModelMappings,
		Enabled:            settings.Enabled,
		HasAPIKey:          settings.UpstreamAPIKey != "",
		CreatedAt:          settings.CreatedAt,
		UpdatedAt:          settings.UpdatedAt,
	}, nil
}

func (s *AmpService) TestConnection(userID string) (*model.TestConnectionResponse, error) {
	settings, err := s.settingsRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	if settings == nil || settings.UpstreamURL == "" {
		return &model.TestConnectionResponse{
			Success: false,
			Message: "未配置 Upstream URL",
		}, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", settings.UpstreamURL, nil)
	if err != nil {
		return &model.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("创建请求失败: %v", err),
		}, nil
	}

	if settings.UpstreamAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+settings.UpstreamAPIKey)
		req.Header.Set("X-Api-Key", settings.UpstreamAPIKey)
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &model.TestConnectionResponse{
			Success:   false,
			Message:   fmt.Sprintf("连接失败: %v", err),
			LatencyMs: latency,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &model.TestConnectionResponse{
			Success:   true,
			Message:   fmt.Sprintf("连接成功 (HTTP %d)", resp.StatusCode),
			LatencyMs: latency,
		}, nil
	}

	if resp.StatusCode == 401 {
		if settings.UpstreamAPIKey == "" {
			return &model.TestConnectionResponse{
				Success:   true,
				Message:   fmt.Sprintf("上游可达，但需要 API Key 认证 (HTTP %d)", resp.StatusCode),
				LatencyMs: latency,
			}, nil
		}
		return &model.TestConnectionResponse{
			Success:   false,
			Message:   fmt.Sprintf("API Key 无效或已过期 (HTTP %d)", resp.StatusCode),
			LatencyMs: latency,
		}, nil
	}

	return &model.TestConnectionResponse{
		Success:   false,
		Message:   fmt.Sprintf("上游返回错误: HTTP %d", resp.StatusCode),
		LatencyMs: latency,
	}, nil
}

func (s *AmpService) CreateAPIKey(userID string, req *model.CreateAPIKeyRequest) (*model.CreateAPIKeyResponse, error) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, err
	}
	rawKey := base64.RawURLEncoding.EncodeToString(keyBytes)

	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	prefix := rawKey[:8]

	apiKey := &model.UserAPIKey{
		UserID:  userID,
		Name:    req.Name,
		Prefix:  prefix,
		KeyHash: keyHash,
	}

	if err := s.apiKeyRepo.Create(apiKey); err != nil {
		return nil, err
	}

	return &model.CreateAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Prefix:    apiKey.Prefix,
		APIKey:    rawKey,
		CreatedAt: apiKey.CreatedAt,
		Message:   "API Key 创建成功，请妥善保存，此 Key 仅显示一次",
	}, nil
}

func (s *AmpService) ListAPIKeys(userID string) ([]*model.APIKeyListItem, error) {
	keys, err := s.apiKeyRepo.ListByUserID(userID)
	if err != nil {
		return nil, err
	}

	items := make([]*model.APIKeyListItem, len(keys))
	for i, k := range keys {
		items[i] = &model.APIKeyListItem{
			ID:        k.ID,
			Name:      k.Name,
			Prefix:    k.Prefix,
			CreatedAt: k.CreatedAt,
			RevokedAt: k.RevokedAt,
			LastUsed:  k.LastUsed,
			IsActive:  k.RevokedAt == nil,
		}
	}
	return items, nil
}

func (s *AmpService) RevokeAPIKey(userID, keyID string) error {
	key, err := s.apiKeyRepo.GetByID(keyID)
	if err != nil {
		return err
	}
	if key == nil {
		return ErrAPIKeyNotFound
	}
	if key.UserID != userID {
		return ErrNotOwner
	}
	if key.RevokedAt != nil {
		return ErrAPIKeyRevoked
	}
	return s.apiKeyRepo.Revoke(keyID)
}

func (s *AmpService) GetBootstrap(userID string) (*model.BootstrapResponse, error) {
	cfg := config.Get()

	settings, err := s.settingsRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	hasAPIKey, err := s.apiKeyRepo.HasActiveByUserID(userID)
	if err != nil {
		return nil, err
	}

	proxyBaseURL := cfg.ProxyBaseURL
	if proxyBaseURL == "" {
		proxyBaseURL = "http://localhost:" + cfg.ServerPort
	}

	configExample := `# 在 Amp 客户端中配置:
# 1. 创建一个 API Key
# 2. 设置环境变量:
export AMP_URL="` + proxyBaseURL + `"
export AMP_API_KEY="{YOUR_API_KEY}"
`

	return &model.BootstrapResponse{
		ProxyBaseURL:  proxyBaseURL,
		ConfigExample: configExample,
		HasSettings:   settings != nil && settings.UpstreamAPIKey != "",
		HasAPIKey:     hasAPIKey,
	}, nil
}

func (s *AmpService) GetSettingsInternal(userID string) (*model.AmpSettings, error) {
	return s.settingsRepo.GetByUserID(userID)
}

func (s *AmpService) ValidateAPIKey(rawKey string) (*model.UserAPIKey, error) {
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	key, err := s.apiKeyRepo.GetByKeyHash(keyHash)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrAPIKeyNotFound
	}
	if key.RevokedAt != nil {
		return nil, ErrAPIKeyRevoked
	}

	go s.apiKeyRepo.UpdateLastUsed(key.ID)

	return key, nil
}
