package main

import (
	"client-monitor/config"
	"client-monitor/database"
	"client-monitor/handlers"
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
		api.DELETE("/clients/:client_id", handlers.DeleteClient)
		api.GET("/clients/orders", handlers.GetClientOrders)
		api.PUT("/clients/order", handlers.UpdateClientOrder)
		api.PUT("/clients/orders", handlers.UpdateAllClientOrders)
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
