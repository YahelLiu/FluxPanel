package main

import (
	"client-monitor/config"
	"client-monitor/database"
	"client-monitor/handlers"
	"client-monitor/notify"
	"log"

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
		wecom := api.Group("/wecom")
		{
			wecom.GET("/callback", handlers.HandleWeComVerify)
			wecom.POST("/callback", handlers.HandleWeComCallback)
			wecom.GET("/config", handlers.GetWeComConfig)
			wecom.PUT("/config", handlers.UpdateWeComConfig)
			wecom.POST("/test", handlers.HandleWeComTest)
			wecom.POST("/chat", handlers.HandleWeComChat)
		}

		// AI Assistant routes
		assistant := api.Group("/assistant")
		{
			assistant.GET("/llm", handlers.GetLLMConfig)
			assistant.PUT("/llm", handlers.UpdateLLMConfig)
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
