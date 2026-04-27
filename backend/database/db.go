package database

import (
	"client-monitor/config"
	"client-monitor/models"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// Auto migrate
	if err = DB.AutoMigrate(
		&models.Event{},
		&models.ClientOrder{},
		&models.NotificationChannel{},
		&models.NotificationRule{},
		&models.NotificationLog{},
		&models.AlertThreshold{},
		&models.AlertRecord{},
		&models.WeatherConfig{},
		&models.WeatherSchedule{},
		&models.WeatherRecord{},
		// AI Assistant models
		&models.AIUser{},
		&models.Conversation{},
		&models.Memory{},
		&models.Todo{},
		&models.Reminder{},
		&models.LLMConfig{},
		&models.WeComConfig{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}
