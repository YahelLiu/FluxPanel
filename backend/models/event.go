package models

import (
	"time"

	"gorm.io/datatypes"
)

type Event struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ClientID  string         `gorm:"index" json:"client_id"`
	EventType string         `gorm:"index" json:"event_type"`
	Data      datatypes.JSON `gorm:"type:jsonb" json:"data"`
	Status    string         `gorm:"index" json:"status"` // success, error, warning
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
}

type ReportRequest struct {
	ClientID  string                 `json:"client_id" binding:"required"`
	EventType string                 `json:"event_type" binding:"required"`
	Data      map[string]interface{} `json:"data"`
	Status    string                 `json:"status"` // default: success
}

type SummaryResponse struct {
	OnlineClients   int64            `json:"online_clients"`
	TodayEvents     int64            `json:"today_events"`
	TodayErrors     int64            `json:"today_errors"`
	EventTypeCounts map[string]int64 `json:"event_type_counts"`
	StatusCounts    map[string]int64 `json:"status_counts"`
}

type EventFilter struct {
	ClientID  string `form:"client_id"`
	Status    string `form:"status"`
	EventType string `form:"event_type"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type EventListResponse struct {
	Total  int64   `json:"total"`
	Events []Event `json:"events"`
}
