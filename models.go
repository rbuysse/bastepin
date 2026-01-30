package main

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint      `gorm:"primaryKey"`
	Username     string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	Pastes       []Paste   `gorm:"foreignKey:UserID"`
}

type Paste struct {
	ID          string         `gorm:"primaryKey"`
	Title       string         `gorm:"default:''"`
	Content     string         `gorm:"not null"`
	ContentHash string         `gorm:"index;not null"`
	Language    string         `gorm:"default:'text'"`
	IsPrivate   bool           `gorm:"default:false"`
	Unlisted    bool           `gorm:"default:false;index"`
	ExpiresAt   *time.Time     `gorm:"index"` // nil = never expires
	UserID      *uint          `gorm:"index"`
	User        *User          `gorm:"foreignKey:UserID"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type Session struct {
	ID        string    `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	User      User      `gorm:"foreignKey:UserID"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type APIKey struct {
	ID        uint       `gorm:"primaryKey"`
	Key       string     `gorm:"uniqueIndex;not null"` // The actual API key
	Name      string     `gorm:"not null"`             // User-friendly name
	UserID    uint       `gorm:"not null;index"`
	User      User       `gorm:"foreignKey:UserID"`
	ExpiresAt *time.Time `gorm:"index"` // nil = never expires
	LastUsed  *time.Time
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type Admin struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"uniqueIndex;not null"`
	User      User      `gorm:"foreignKey:UserID"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
