package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"ampmanager/internal/database"
	"ampmanager/internal/model"
)

type RequestLogRepository struct{}

func NewRequestLogRepository() *RequestLogRepository {
	return &RequestLogRepository{}
}

// ListParams 查询参数
type ListParams struct {
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
func (r *RequestLogRepository) List(params ListParams) ([]model.RequestLog, int64, error) {
	db := database.GetDB()

	// 构建 WHERE 条件
	conditions := []string{"1=1"}
	args := []interface{}{}

	if params.UserID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, params.UserID)
	}
	if params.APIKeyID != "" {
		conditions = append(conditions, "api_key_id = ?")
		args = append(args, params.APIKeyID)
	}
	if params.Model != "" {
		conditions = append(conditions, "(original_model = ? OR mapped_model = ?)")
		args = append(args, params.Model, params.Model)
	}
	if params.StatusCode != nil {
		conditions = append(conditions, "status_code = ?")
		args = append(args, *params.StatusCode)
	}
	if params.IsStreaming != nil {
		val := 0
		if *params.IsStreaming {
			val = 1
		}
		conditions = append(conditions, "is_streaming = ?")
		args = append(args, val)
	}
	if params.From != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *params.From)
	}
	if params.To != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, *params.To)
	}

	whereClause := strings.Join(conditions, " AND ")

	// 查询总数
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM request_logs WHERE %s", whereClause)
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 分页
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	offset := (params.Page - 1) * params.PageSize

	// 查询数据
	query := fmt.Sprintf(`
		SELECT id, created_at, updated_at, status, user_id, api_key_id, original_model, mapped_model,
		       provider, channel_id, endpoint, method, path, status_code, latency_ms,
		       is_streaming, input_tokens, output_tokens, cache_read_input_tokens,
		       cache_creation_input_tokens, error_type, request_id
		FROM request_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, params.PageSize, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.RequestLog
	for rows.Next() {
		var log model.RequestLog
		var createdAt time.Time
		var updatedAt sql.NullTime
		var status sql.NullString
		var isStreaming int
		var originalModel, mappedModel, provider, channelID, endpoint, errorType, requestID sql.NullString
		var inputTokens, outputTokens, cacheRead, cacheCreation sql.NullInt64

		err := rows.Scan(
			&log.ID, &createdAt, &updatedAt, &status, &log.UserID, &log.APIKeyID,
			&originalModel, &mappedModel, &provider, &channelID, &endpoint,
			&log.Method, &log.Path, &log.StatusCode, &log.LatencyMs,
			&isStreaming, &inputTokens, &outputTokens, &cacheRead, &cacheCreation,
			&errorType, &requestID,
		)
		if err != nil {
			return nil, 0, err
		}

		log.CreatedAt = createdAt.Format(time.RFC3339)
		log.IsStreaming = isStreaming == 1

		if updatedAt.Valid {
			formatted := updatedAt.Time.Format(time.RFC3339)
			log.UpdatedAt = &formatted
		}
		if status.Valid {
			log.Status = model.RequestLogStatus(status.String)
		} else {
			log.Status = model.RequestLogStatusSuccess // 默认为 success（兼容旧数据）
		}

		if originalModel.Valid {
			log.OriginalModel = &originalModel.String
		}
		if mappedModel.Valid {
			log.MappedModel = &mappedModel.String
		}
		if provider.Valid {
			log.Provider = &provider.String
		}
		if channelID.Valid {
			log.ChannelID = &channelID.String
		}
		if endpoint.Valid {
			log.Endpoint = &endpoint.String
		}
		if errorType.Valid {
			log.ErrorType = &errorType.String
		}
		if requestID.Valid {
			log.RequestID = &requestID.String
		}
		if inputTokens.Valid {
			v := int(inputTokens.Int64)
			log.InputTokens = &v
		}
		if outputTokens.Valid {
			v := int(outputTokens.Int64)
			log.OutputTokens = &v
		}
		if cacheRead.Valid {
			v := int(cacheRead.Int64)
			log.CacheReadInputTokens = &v
		}
		if cacheCreation.Valid {
			v := int(cacheCreation.Int64)
			log.CacheCreationInputTokens = &v
		}

		logs = append(logs, log)
	}

	return logs, total, rows.Err()
}

// GetUsageSummary 获取用量统计
func (r *RequestLogRepository) GetUsageSummary(userID string, from, to *time.Time, groupBy string) ([]model.UsageSummary, error) {
	db := database.GetDB()

	var groupColumn string
	switch groupBy {
	case "day":
		groupColumn = "substr(created_at, 1, 10)"
	case "model":
		groupColumn = "COALESCE(mapped_model, original_model, 'unknown')"
	case "apiKey":
		groupColumn = "api_key_id"
	default:
		groupColumn = "date(created_at)"
	}

	conditions := []string{"user_id = ?"}
	args := []interface{}{userID}

	if from != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *from)
	}
	if to != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, *to)
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT 
			%s as group_key,
			COALESCE(SUM(input_tokens), 0) as input_tokens_sum,
			COALESCE(SUM(output_tokens), 0) as output_tokens_sum,
			COALESCE(SUM(cache_read_input_tokens), 0) as cache_read_sum,
			COALESCE(SUM(cache_creation_input_tokens), 0) as cache_creation_sum,
			COUNT(*) as request_count,
			SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as error_count
		FROM request_logs
		WHERE %s
		GROUP BY %s
		ORDER BY %s DESC
		LIMIT 100
	`, groupColumn, whereClause, groupColumn, groupColumn)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []model.UsageSummary
	for rows.Next() {
		var s model.UsageSummary
		err := rows.Scan(
			&s.GroupKey,
			&s.InputTokensSum,
			&s.OutputTokensSum,
			&s.CacheReadInputTokensSum,
			&s.CacheCreationInputTokensSum,
			&s.RequestCount,
			&s.ErrorCount,
		)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}

	return summaries, rows.Err()
}

// GetByID 获取单条日志
func (r *RequestLogRepository) GetByID(id string) (*model.RequestLog, error) {
	db := database.GetDB()

	var log model.RequestLog
	var createdAt time.Time
	var updatedAt sql.NullTime
	var status sql.NullString
	var isStreaming int
	var originalModel, mappedModel, provider, channelID, endpoint, errorType, requestID sql.NullString
	var inputTokens, outputTokens, cacheRead, cacheCreation sql.NullInt64

	err := db.QueryRow(`
		SELECT id, created_at, updated_at, status, user_id, api_key_id, original_model, mapped_model,
		       provider, channel_id, endpoint, method, path, status_code, latency_ms,
		       is_streaming, input_tokens, output_tokens, cache_read_input_tokens,
		       cache_creation_input_tokens, error_type, request_id
		FROM request_logs WHERE id = ?
	`, id).Scan(
		&log.ID, &createdAt, &updatedAt, &status, &log.UserID, &log.APIKeyID,
		&originalModel, &mappedModel, &provider, &channelID, &endpoint,
		&log.Method, &log.Path, &log.StatusCode, &log.LatencyMs,
		&isStreaming, &inputTokens, &outputTokens, &cacheRead, &cacheCreation,
		&errorType, &requestID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	log.CreatedAt = createdAt.Format(time.RFC3339)
	log.IsStreaming = isStreaming == 1

	if updatedAt.Valid {
		formatted := updatedAt.Time.Format(time.RFC3339)
		log.UpdatedAt = &formatted
	}
	if status.Valid {
		log.Status = model.RequestLogStatus(status.String)
	} else {
		log.Status = model.RequestLogStatusSuccess
	}

	if originalModel.Valid {
		log.OriginalModel = &originalModel.String
	}
	if mappedModel.Valid {
		log.MappedModel = &mappedModel.String
	}
	if provider.Valid {
		log.Provider = &provider.String
	}
	if channelID.Valid {
		log.ChannelID = &channelID.String
	}
	if endpoint.Valid {
		log.Endpoint = &endpoint.String
	}
	if errorType.Valid {
		log.ErrorType = &errorType.String
	}
	if requestID.Valid {
		log.RequestID = &requestID.String
	}
	if inputTokens.Valid {
		v := int(inputTokens.Int64)
		log.InputTokens = &v
	}
	if outputTokens.Valid {
		v := int(outputTokens.Int64)
		log.OutputTokens = &v
	}
	if cacheRead.Valid {
		v := int(cacheRead.Int64)
		log.CacheReadInputTokens = &v
	}
	if cacheCreation.Valid {
		v := int(cacheCreation.Int64)
		log.CacheCreationInputTokens = &v
	}

	return &log, nil
}
