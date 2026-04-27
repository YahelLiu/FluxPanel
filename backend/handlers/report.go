package handlers

import (
	"client-monitor/database"
	"client-monitor/models"
	"encoding/json"
	"net/http"
	"time"

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
	BroadcastEvent(event)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      event.ID,
	})
}

// Summary handles GET /api/summary - get summary statistics
func Summary(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	fiveMinAgo := now.Add(-5 * time.Minute)

	var onlineClients int64
	database.DB.Model(&models.Event{}).
		Where("created_at > ?", fiveMinAgo).
		Distinct("client_id").
		Count(&onlineClients)

	var todayEvents int64
	database.DB.Model(&models.Event{}).
		Where("created_at > ?", today).
		Count(&todayEvents)

	var todayErrors int64
	database.DB.Model(&models.Event{}).
		Where("created_at > ? AND status = ?", today, "error").
		Count(&todayErrors)

	// Event type counts
	type CountResult struct {
		Key   string
		Count int64
	}
	var eventTypeResults []CountResult
	database.DB.Model(&models.Event{}).
		Select("event_type as key, count(*) as count").
		Where("created_at > ?", today).
		Group("event_type").
		Scan(&eventTypeResults)

	eventTypeCounts := make(map[string]int64)
	for _, r := range eventTypeResults {
		eventTypeCounts[r.Key] = r.Count
	}

	// Status counts
	var statusResults []CountResult
	database.DB.Model(&models.Event{}).
		Select("status as key, count(*) as count").
		Where("created_at > ?", today).
		Group("status").
		Scan(&statusResults)

	statusCounts := make(map[string]int64)
	for _, r := range statusResults {
		statusCounts[r.Key] = r.Count
	}

	c.JSON(http.StatusOK, models.SummaryResponse{
		OnlineClients:   onlineClients,
		TodayEvents:     todayEvents,
		TodayErrors:     todayErrors,
		EventTypeCounts: eventTypeCounts,
		StatusCounts:    statusCounts,
	})
}

// Events handles GET /api/events - get event list with filters
func Events(c *gin.Context) {
	var filter models.EventFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	query := database.DB.Model(&models.Event{})

	if filter.ClientID != "" {
		query = query.Where("client_id = ?", filter.ClientID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}

	var total int64
	query.Count(&total)

	var events []models.Event
	offset := (filter.Page - 1) * filter.PageSize
	query.Order("created_at DESC").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&events)

	c.JSON(http.StatusOK, models.EventListResponse{
		Total:  total,
		Events: events,
	})
}

// HourlyStats handles GET /api/stats/hourly - hourly event counts for charts
func HourlyStats(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	type HourlyResult struct {
		Hour  int   `json:"hour"`
		Count int64 `json:"count"`
	}

	var results []HourlyResult
	database.DB.Model(&models.Event{}).
		Select("EXTRACT(HOUR FROM created_at) as hour, count(*) as count").
		Where("created_at > ?", today).
		Group("hour").
		Order("hour").
		Scan(&results)

	c.JSON(http.StatusOK, results)
}

// ClientStats handles GET /api/stats/clients - client event counts for charts
func ClientStats(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	type ClientResult struct {
		ClientID string `json:"client_id"`
		Count    int64  `json:"count"`
	}

	var results []ClientResult
	database.DB.Model(&models.Event{}).
		Select("client_id, count(*) as count").
		Where("created_at > ?", today).
		Group("client_id").
		Order("count DESC").
		Limit(10).
		Scan(&results)

	c.JSON(http.StatusOK, results)
}

// DeleteClient handles DELETE /api/clients/:client_id - delete all events for a client
func DeleteClient(c *gin.Context) {
	clientID := c.Param("client_id")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id is required"})
		return
	}

	result := database.DB.Where("client_id = ?", clientID).Delete(&models.Event{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete client"})
		return
	}

	// Also delete client order
	database.DB.Where("client_id = ?", clientID).Delete(&models.ClientOrder{})

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"deleted":     result.RowsAffected,
		"message":     "Client deleted successfully",
	})
}

// GetClientOrders handles GET /api/clients/orders - get all client sort orders
func GetClientOrders(c *gin.Context) {
	var orders []models.ClientOrder
	database.DB.Order("sort_order ASC").Find(&orders)

	ordersMap := make(map[string]int)
	for _, o := range orders {
		ordersMap[o.ClientID] = o.SortOrder
	}

	c.JSON(http.StatusOK, ordersMap)
}

// UpdateClientOrder handles PUT /api/clients/order - update client sort order
func UpdateClientOrder(c *gin.Context) {
	var req struct {
		ClientID string `json:"client_id" binding:"required"`
		SortOrder int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order := models.ClientOrder{
		ClientID: req.ClientID,
		SortOrder: req.SortOrder,
	}

	if err := database.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateAllClientOrders handles PUT /api/clients/orders - update all client sort orders
func UpdateAllClientOrders(c *gin.Context) {
	var req struct {
		Orders []models.ClientOrder `json:"orders" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update each order
	for _, o := range req.Orders {
		database.DB.Save(&o)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
