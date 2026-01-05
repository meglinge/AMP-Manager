package service

import (
	"time"

	"ampmanager/internal/model"
	"ampmanager/internal/repository"
)

type RequestLogService struct {
	repo *repository.RequestLogRepository
}

func NewRequestLogService() *RequestLogService {
	return &RequestLogService{
		repo: repository.NewRequestLogRepository(),
	}
}

// ListParams 列表查询参数
type ListRequestLogsParams struct {
	UserID      string
	APIKeyID    string
	Model       string
	StatusCode  *int
	IsStreaming *bool
	From        *time.Time
	To          *time.Time
	Page        int
	PageSize    int
}

// List 查询请求日志列表
func (s *RequestLogService) List(params ListRequestLogsParams) (*model.RequestLogListResponse, error) {
	repoParams := repository.ListParams{
		UserID:      params.UserID,
		APIKeyID:    params.APIKeyID,
		Model:       params.Model,
		StatusCode:  params.StatusCode,
		IsStreaming: params.IsStreaming,
		From:        params.From,
		To:          params.To,
		Page:        params.Page,
		PageSize:    params.PageSize,
	}

	logs, total, err := s.repo.List(repoParams)
	if err != nil {
		return nil, err
	}

	return &model.RequestLogListResponse{
		Items:    logs,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

// GetUsageSummary 获取用量统计
func (s *RequestLogService) GetUsageSummary(userID string, from, to *time.Time, groupBy string) (*model.UsageSummaryResponse, error) {
	summaries, err := s.repo.GetUsageSummary(userID, from, to, groupBy)
	if err != nil {
		return nil, err
	}

	return &model.UsageSummaryResponse{
		Items: summaries,
	}, nil
}

// GetByID 获取单条日志
func (s *RequestLogService) GetByID(id, userID string) (*model.RequestLog, error) {
	log, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 验证所有权
	if log != nil && log.UserID != userID {
		return nil, nil
	}

	return log, nil
}
