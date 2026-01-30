package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func initDatabase(dbPath string, debug bool) error {
	var err error

	gormConfig := &gorm.Config{}
	if !debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	} else {
		// In debug mode, only log warnings and errors, not "record not found" info messages
		gormConfig.Logger = logger.Default.LogMode(logger.Warn)
	}

	db, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&User{}, &Paste{}, &Session{}, &APIKey{}, &Admin{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	if debug {
		log.Println("Database initialized successfully")
	}

	return nil
}

func cleanExpiredSessions() error {
	return db.Where("expires_at < ?", time.Now()).Delete(&Session{}).Error
}
