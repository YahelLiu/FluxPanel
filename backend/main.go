package main

import (
	"context"
	"client-monitor/config"
	"client-monitor/database"
	"client-monitor/handlers"
	"client-monitor/models"
	"client-monitor/notify"
	"client-monitor/wecom"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	log.Println("Database connected successfully")

	// Create root context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Initialize WebSocket
	handlers.InitWebSocket()

	// Initialize notification service
	notify.GetService()
	notify.GetAlertService()
	log.Println("Notification service initialized")

	// Initialize weather service
	notify.GetWeatherService().Start()
	log.Println("Weather service initialized")

	// Initialize reminder service
	notify.GetReminderService().Start()
	log.Println("Reminder service initialized")

	// 设置提醒发送回调，通过 WebSocket 推送
	notify.SetSendReminderCallback(func(reminder *models.Reminder) {
		handlers.SendReminderViaWebSocket(reminder)
	})
	log.Println("Reminder WebSocket callback registered")

	// Start WeCom iLink monitor if wechat_ilink channel exists and is logged in
	if wecom.HasWechatILinkChannel() {
		go startWeComMonitor(ctx)
		log.Println("WeCom iLink monitor started")
	} else {
		log.Println("WeCom iLink not configured, skipping monitor startup")
	}

	// 定期检查并尝试启动 Monitor（支持热加载，登录后自动启动）
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		monitorStarted := wecom.HasWechatILinkChannel()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hasChannel := wecom.HasWechatILinkChannel()
				if hasChannel && !monitorStarted {
					log.Println("[wecom] Detected new login, starting monitor...")
					go startWeComMonitor(ctx)
					monitorStarted = true
				}
				// 如果用户登出，重置状态以便下次登录能重新启动
				if !hasChannel {
					monitorStarted = false
				}
			}
		}
	}()

	// Setup Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	// API routes
	api := r.Group("/api")
	{
		api.POST("/report", handlers.Report)
		api.GET("/summary", handlers.Summary)
		api.GET("/events", handlers.Events)
		api.GET("/stats/hourly", handlers.HourlyStats)
		api.GET("/stats/clients", handlers.ClientStats)
		api.GET("/clients/latest", handlers.LatestClients)
		api.DELETE("/clients/:client_id", handlers.DeleteClient)
		api.GET("/clients/orders", handlers.GetClientOrders)
		api.PUT("/clients/order", handlers.UpdateClientOrder)
		api.PUT("/clients/orders", handlers.UpdateAllClientOrders)
		api.PUT("/clients/:client_id/weather", handlers.UpdateClientWeather)

		// Notification routes
		notifications := api.Group("/notifications")
		{
			// Channels
			notifications.GET("/channels", handlers.ListChannels)
			notifications.GET("/channels/:id", handlers.GetChannel)
			notifications.POST("/channels", handlers.CreateChannel)
			notifications.PUT("/channels/:id", handlers.UpdateChannel)
			notifications.DELETE("/channels/:id", handlers.DeleteChannel)
			notifications.POST("/channels/:id/test", handlers.TestChannel)

			// Wechat iLink 登录
			notifications.GET("/channels/wechat-ilink", handlers.GetWechatILinkChannel)
			notifications.GET("/channels/wechat-ilink/qrcode", handlers.GetWechatILinkQRCode)
			notifications.GET("/channels/wechat-ilink/status", handlers.GetWechatILinkStatus)
			notifications.POST("/channels/wechat-ilink/logout", handlers.LogoutWechatILink)

			// Rules
			notifications.GET("/rules", handlers.ListRules)
			notifications.POST("/rules", handlers.CreateRule)
			notifications.PUT("/rules/:id", handlers.UpdateRule)
			notifications.DELETE("/rules/:id", handlers.DeleteRule)

			// Logs
			notifications.GET("/logs", handlers.ListLogs)
		}

		// Alert routes
		alerts := api.Group("/alerts")
		{
			// Thresholds
			alerts.GET("/thresholds", handlers.ListAlertThresholds)
			alerts.POST("/thresholds", handlers.CreateAlertThreshold)
			alerts.PUT("/thresholds/:id", handlers.UpdateAlertThreshold)
			alerts.DELETE("/thresholds/:id", handlers.DeleteAlertThreshold)
			alerts.PUT("/thresholds/:id/toggle", handlers.ToggleAlertThreshold)
			alerts.POST("/thresholds/:id/test", handlers.TestAlertThreshold)

			// Records
			alerts.GET("/records", handlers.ListAlertRecords)
			alerts.GET("/active", handlers.GetActiveAlerts)
			alerts.PUT("/records/:id/resolve", handlers.ResolveAlert)
			alerts.DELETE("/records/:id", handlers.DeleteAlertRecord)
		}

		// Weather routes
		weather := api.Group("/weather")
		{
			weather.GET("/config", handlers.GetWeatherConfig)
			weather.PUT("/config", handlers.UpdateWeatherConfig)
			weather.POST("/test", handlers.TestWeatherConfig)
			weather.GET("/schedules", handlers.GetWeatherSchedules)
			weather.PUT("/schedules", handlers.UpdateWeatherSchedules)
			weather.GET("/records", handlers.GetWeatherRecords)
			weather.POST("/send", handlers.SendWeatherNow)
		}

		// WeCom routes
		wecomGroup := api.Group("/wecom")
		{
			// Login (iLink)
			wecomGroup.GET("/login/qrcode", handlers.GetLoginQRCode)
			wecomGroup.GET("/login/status", handlers.GetLoginStatus)
			wecomGroup.GET("/status", handlers.GetWeComStatus)
			wecomGroup.DELETE("/session", handlers.Logout)

			// Monitor status
			wecomGroup.GET("/monitor/status", handlers.GetMonitorStatus)

			// Test and chat
			wecomGroup.POST("/test", handlers.HandleWeComTest)
			wecomGroup.POST("/chat", handlers.HandleWeComChat)

			// Legacy config (keep for compatibility)
			wecomGroup.GET("/config", handlers.GetWeComConfig)
			wecomGroup.PUT("/config", handlers.UpdateWeComConfig)

			// Reminders
			wecomGroup.GET("/reminders/pending", handlers.GetPendingReminders)
			wecomGroup.POST("/reminders/:id/sent", handlers.MarkReminderSent)
		}

		// AI Assistant routes
		assistant := api.Group("/assistant")
		{
			assistant.GET("/llm", handlers.GetLLMConfig)
			assistant.PUT("/llm", handlers.UpdateLLMConfig)
			assistant.POST("/llm/test", handlers.TestLLM)
			assistant.GET("/users", handlers.GetAIUsers)
			assistant.GET("/conversations", handlers.GetConversations)
			assistant.GET("/todos", handlers.GetTodos)
			assistant.POST("/todos", handlers.CreateTodo)
			assistant.PUT("/todos/:id", handlers.UpdateTodo)
			assistant.DELETE("/todos/:id", handlers.DeleteTodo)
			assistant.GET("/memories", handlers.GetMemories)
			assistant.DELETE("/memories/:id", handlers.DeleteMemory)
			assistant.GET("/reminders", handlers.GetReminders)
		}
	}

	// WebSocket route
	r.GET("/ws", handlers.HandleWebSocket)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	log.Printf("Server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// startWeComMonitor 启动 WeCom iLink 监听器
func startWeComMonitor(ctx context.Context) {
	monitor, err := wecom.NewMonitor()
	if err != nil {
		log.Printf("[wecom] Failed to create monitor: %v", err)
		return
	}

	monitor.RunWithRestart(ctx)
}
