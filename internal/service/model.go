package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ampmanager/internal/model"
	"ampmanager/internal/repository"

	log "github.com/sirupsen/logrus"
)

type ModelService struct {
	channelRepo      *repository.ChannelRepository
	channelModelRepo *repository.ChannelModelRepository
}

func NewModelService() *ModelService {
	return &ModelService{
		channelRepo:      repository.NewChannelRepository(),
		channelModelRepo: repository.NewChannelModelRepository(),
	}
}

func (s *ModelService) FetchAndSaveModels(channelID string) (int, error) {
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return 0, err
	}
	if channel == nil {
		return 0, fmt.Errorf("渠道不存在")
	}

	models, err := s.fetchModelsFromProvider(channel)
	if err != nil {
		return 0, err
	}

	filteredModels := s.filterModelsByType(channel.Type, models)

	channelModels := make([]model.ChannelModel2, len(filteredModels))
	for i, m := range filteredModels {
		channelModels[i] = model.ChannelModel2{
			ChannelID:   channelID,
			ModelID:     m.ID,
			DisplayName: m.DisplayName,
		}
	}

	if err := s.channelModelRepo.ReplaceModels(channelID, channelModels); err != nil {
		return 0, err
	}

	return len(channelModels), nil
}

type fetchedModel struct {
	ID          string
	DisplayName string
}

func (s *ModelService) fetchModelsFromProvider(channel *model.Channel) ([]fetchedModel, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	var (
		url string
		req *http.Request
		err error
	)

	switch channel.Type {
	case model.ChannelTypeOpenAI:
		url = strings.TrimSuffix(channel.BaseURL, "/") + "/v1/models"
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+channel.APIKey)

	case model.ChannelTypeClaude:
		url = strings.TrimSuffix(channel.BaseURL, "/") + "/v1/models"
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-api-key", channel.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")

	case model.ChannelTypeGemini:
		url = strings.TrimSuffix(channel.BaseURL, "/") + "/v1beta/models?key=" + channel.APIKey
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-goog-api-key", channel.APIKey)

	default:
		return nil, fmt.Errorf("不支持的渠道类型: %s", channel.Type)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败 HTTP %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseModelsResponse(channel.Type, bodyBytes)
}

func (s *ModelService) parseModelsResponse(channelType model.ChannelType, body []byte) ([]fetchedModel, error) {
	switch channelType {
	case model.ChannelTypeOpenAI:
		var resp struct {
			Data []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		models := make([]fetchedModel, len(resp.Data))
		for i, m := range resp.Data {
			models[i] = fetchedModel{ID: m.ID, DisplayName: m.ID}
		}
		return models, nil

	case model.ChannelTypeClaude:
		var resp struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
				Type        string `json:"type"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		models := make([]fetchedModel, len(resp.Data))
		for i, m := range resp.Data {
			displayName := m.DisplayName
			if displayName == "" {
				displayName = m.ID
			}
			models[i] = fetchedModel{ID: m.ID, DisplayName: displayName}
		}
		return models, nil

	case model.ChannelTypeGemini:
		var resp struct {
			Models []struct {
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		models := make([]fetchedModel, len(resp.Models))
		for i, m := range resp.Models {
			modelID := strings.TrimPrefix(m.Name, "models/")
			displayName := m.DisplayName
			if displayName == "" {
				displayName = modelID
			}
			models[i] = fetchedModel{ID: modelID, DisplayName: displayName}
		}
		return models, nil

	default:
		return nil, fmt.Errorf("不支持的渠道类型")
	}
}

func (s *ModelService) filterModelsByType(channelType model.ChannelType, models []fetchedModel) []fetchedModel {
	var filtered []fetchedModel

	for _, m := range models {
		idLower := strings.ToLower(m.ID)

		switch channelType {
		case model.ChannelTypeOpenAI:
			if strings.HasPrefix(idLower, "gpt") ||
				strings.HasPrefix(idLower, "o1") ||
				strings.HasPrefix(idLower, "o3") ||
				strings.HasPrefix(idLower, "o4") ||
				strings.HasPrefix(idLower, "chatgpt") {
				filtered = append(filtered, m)
			}

		case model.ChannelTypeClaude:
			if strings.HasPrefix(idLower, "claude") {
				filtered = append(filtered, m)
			}

		case model.ChannelTypeGemini:
			if strings.HasPrefix(idLower, "gemini") {
				filtered = append(filtered, m)
			}
		}
	}

	log.Infof("模型过滤: 类型=%s, 总数=%d, 过滤后=%d", channelType, len(models), len(filtered))
	return filtered
}

func (s *ModelService) GetModelsByChannelID(channelID string) ([]*model.ChannelModel2, error) {
	return s.channelModelRepo.GetByChannelID(channelID)
}

func (s *ModelService) ListAllAvailableModels() ([]*model.AvailableModel, error) {
	return s.channelModelRepo.ListAllWithChannel()
}

func (s *ModelService) FetchAllChannelsModels() (map[string]int, error) {
	channels, err := s.channelRepo.ListEnabled()
	if err != nil {
		return nil, err
	}

	results := make(map[string]int)
	for _, ch := range channels {
		count, err := s.FetchAndSaveModels(ch.ID)
		if err != nil {
			log.Warnf("获取渠道 %s 模型失败: %v", ch.Name, err)
			results[ch.Name] = -1
		} else {
			results[ch.Name] = count
		}
	}

	return results, nil
}
