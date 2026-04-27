package handlers

import (
	"net/http"
	"strconv"
	"time"

	"client-monitor/database"
	"client-monitor/models"

	"github.com/gin-gonic/gin"
)

// Summary GET /api/summary - 获取汇总数据
func Summary(c *gin.Context) {
	var onlineClients int64
	var todayEvents int64
	var todayErrors int64

	today := time.Now().Format("2006-01-02")

	// 获取在线客户端数量（今天有报告的客户端）
	database.DB.Model(&models.Event{}).
		Where("DATE(created_at) = ?", today).
		Distinct("client_id").
		Count(&onlineClients)

	// 获取今天的事件总数
	database.DB.Model(&models.Event{}).
		Where("DATE(created_at) = ?", today).
		Count(&todayEvents)

	// 获取今天的错误数
	database.DB.Model(&models.Event{}).
		Where("DATE(created_at) = ? AND status = ?", today, "error").
		Count(&todayErrors)

	// 获取事件类型统计
	var eventTypeCounts []struct {
		EventType string `json:"event_type"`
		Count     int64  `json:"count"`
	}
	database.DB.Model(&models.Event{}).
		Select("event_type, COUNT(*) as count").
		Where("DATE(created_at) = ?", today).
		Group("event_type").
		Find(&eventTypeCounts)

	eventTypeMap := make(map[string]int64)
	for _, item := range eventTypeCounts {
		eventTypeMap[item.EventType] = item.Count
	}

	// 获取状态统计
	var statusCounts []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	database.DB.Model(&models.Event{}).
		Select("status, COUNT(*) as count").
		Where("DATE(created_at) = ?", today).
		Group("status").
		Find(&statusCounts)

	statusMap := make(map[string]int64)
	for _, item := range statusCounts {
		statusMap[item.Status] = item.Count
	}

	c.JSON(http.StatusOK, models.SummaryResponse{
		OnlineClients:   onlineClients,
		TodayEvents:     todayEvents,
		TodayErrors:     todayErrors,
		EventTypeCounts: eventTypeMap,
		StatusCounts:    statusMap,
	})
}

// Events GET /api/events - 获取事件列表
func Events(c *gin.Context) {
	var filter models.EventFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
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

// HourlyStats GET /api/stats/hourly - 获取每小时统计
func HourlyStats(c *gin.Context) {
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	var stats []struct {
		Hour   int   `json:"hour"`
		Total  int64 `json:"total"`
		Errors int64 `json:"errors"`
	}

	database.DB.Model(&models.Event{}).
		Select("EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as total, SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as errors").
		Where("DATE(created_at) = ?", date).
		Group("hour").
		Order("hour").
		Find(&stats)

	// 填充24小时数据
	result := make([]map[string]interface{}, 24)
	for i := 0; i < 24; i++ {
		result[i] = map[string]interface{}{
			"hour":   i,
			"total":  0,
			"errors": 0,
		}
	}
	for _, s := range stats {
		result[s.Hour]["total"] = s.Total
		result[s.Hour]["errors"] = s.Errors
	}

	c.JSON(http.StatusOK, result)
}

// ClientStats GET /api/stats/clients - 获取客户端统计
func ClientStats(c *gin.Context) {
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	var stats []struct {
		ClientID  string `json:"client_id"`
		Total     int64  `json:"total"`
		Success   int64  `json:"success"`
		Errors    int64  `json:"errors"`
		Warnings  int64  `json:"warnings"`
		LastEvent string `json:"last_event"`
	}

	database.DB.Model(&models.Event{}).
		Select(`client_id, COUNT(*) as total,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as errors,
			SUM(CASE WHEN status = 'warning' THEN 1 ELSE 0 END) as warnings,
			MAX(created_at) as last_event`).
		Where("DATE(created_at) = ?", date).
		Group("client_id").
		Order("total DESC").
		Find(&stats)

	c.JSON(http.StatusOK, stats)
}

// LatestClients GET /api/clients/latest - 获取最近活跃的客户端
func LatestClients(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}

	var clients []struct {
		ClientID  string    `json:"client_id"`
		LastEvent time.Time `json:"last_event"`
		Status    string    `json:"status"`
	}

	database.DB.Model(&models.Event{}).
		Select("client_id, MAX(created_at) as last_event, (SELECT status FROM events e2 WHERE e2.client_id = events.client_id ORDER BY created_at DESC LIMIT 1) as status").
		Group("client_id").
		Order("last_event DESC").
		Limit(limit).
		Find(&clients)

	c.JSON(http.StatusOK, clients)
}

// DeleteClient DELETE /api/clients/:client_id - 删除客户端数据
func DeleteClient(c *gin.Context) {
	clientID := c.Param("client_id")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id required"})
		return
	}

	// 删除客户端的所有事件
	result := database.DB.Where("client_id = ?", clientID).Delete(&models.Event{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// 删除客户端排序配置
	database.DB.Where("client_id = ?", clientID).Delete(&models.ClientOrder{})

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"deleted_count": result.RowsAffected,
	})
}

// GetClientOrders GET /api/clients/orders - 获取客户端排序配置
func GetClientOrders(c *gin.Context) {
	var orders []models.ClientOrder
	database.DB.Order("sort_order, client_id").Find(&orders)

	c.JSON(http.StatusOK, orders)
}

// UpdateClientOrder PUT /api/clients/order - 更新单个客户端排序
func UpdateClientOrder(c *gin.Context) {
	var req struct {
		ClientID  string `json:"client_id" binding:"required"`
		SortOrder int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order models.ClientOrder
	result := database.DB.Where("client_id = ?", req.ClientID).First(&order)
	if result.Error != nil {
		// 创建新记录
		order = models.ClientOrder{
			ClientID:  req.ClientID,
			SortOrder: req.SortOrder,
		}
		if err := database.DB.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// 更新现有记录
		order.SortOrder = req.SortOrder
		if err := database.DB.Save(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, order)
}

// UpdateAllClientOrders PUT /api/clients/orders - 批量更新客户端排序
func UpdateAllClientOrders(c *gin.Context) {
	var req struct {
		Orders []models.ClientOrder `json:"orders" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 使用事务批量更新
	tx := database.DB.Begin()
	for _, order := range req.Orders {
		var existing models.ClientOrder
		result := tx.Where("client_id = ?", order.ClientID).First(&existing)
		if result.Error != nil {
			tx.Create(&models.ClientOrder{
				ClientID:       order.ClientID,
				SortOrder:      order.SortOrder,
				WeatherEnabled: order.WeatherEnabled,
				ChannelID:      order.ChannelID,
			})
		} else {
			existing.SortOrder = order.SortOrder
			existing.WeatherEnabled = order.WeatherEnabled
			existing.ChannelID = order.ChannelID
			tx.Save(&existing)
		}
	}
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateClientWeather PUT /api/clients/:client_id/weather - 更新客户端天气推送配置
func UpdateClientWeather(c *gin.Context) {
	clientID := c.Param("client_id")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id required"})
		return
	}

	var req struct {
		WeatherEnabled bool  `json:"weather_enabled"`
		ChannelID      uint  `json:"channel_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order models.ClientOrder
	result := database.DB.Where("client_id = ?", clientID).First(&order)
	if result.Error != nil {
		// 创建新记录
		order = models.ClientOrder{
			ClientID:       clientID,
			WeatherEnabled: req.WeatherEnabled,
			ChannelID:      req.ChannelID,
		}
		if err := database.DB.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// 更新现有记录
		order.WeatherEnabled = req.WeatherEnabled
		order.ChannelID = req.ChannelID
		if err := database.DB.Save(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, order)
}
