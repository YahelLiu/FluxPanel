package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/notify"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// Report handles POST /api/report - client submits data
func Report(c *gin.Context) {
	var req models.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default status
	if req.Status == "" {
		req.Status = "success"
	}

	// Convert map to datatypes.JSON
	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data format"})
		return
	}

	event := models.Event{
		ClientID:  req.ClientID,
		EventType: req.EventType,
		Data:      datatypes.JSON(dataJSON),
		Status:    req.Status,
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save event"})
		return
	}

	// Broadcast to WebSocket clients
	BroadcastMessage(jsonMarshal(map[string]interface{}{
		"type":  "event",
		"event": event,
	}))

	// Send notification for error/warning events
	if event.Status == "error" || event.Status == "warning" {
		go notify.GetService().SendNotification(nil, event)
	}

	// Check alert thresholds
	go notify.GetAlertService().CheckEvent(event)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"event":   event,
	})
}
