package model

import "time"

type ModelMetadata struct {
	ID                  string    `json:"id"`
	ModelPattern        string    `json:"modelPattern"`
	DisplayName         string    `json:"displayName"`
	ContextLength       int       `json:"contextLength"`
	MaxCompletionTokens int       `json:"maxCompletionTokens"`
	Provider            string    `json:"provider"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type ModelMetadataRequest struct {
	ModelPattern        string `json:"modelPattern" binding:"required,min=1,max=128"`
	DisplayName         string `json:"displayName"`
	ContextLength       int    `json:"contextLength" binding:"required,min=1000"`
	MaxCompletionTokens int    `json:"maxCompletionTokens" binding:"required,min=100"`
	Provider            string `json:"provider"`
}
