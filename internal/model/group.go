package model

import "time"

type Group struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	RateMultiplier float64   `json:"rateMultiplier"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type GroupRequest struct {
	Name           string  `json:"name" binding:"required,min=1,max=64"`
	Description    string  `json:"description" binding:"max=256"`
	RateMultiplier float64 `json:"rateMultiplier"`
}

type GroupResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	RateMultiplier float64   `json:"rateMultiplier"`
	UserCount      int       `json:"userCount"`
	ChannelCount   int       `json:"channelCount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
