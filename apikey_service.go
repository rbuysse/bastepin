package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"gorm.io/gorm"
)

type APIKeyService struct {
	db *gorm.DB
}

func NewAPIKeyService(database *gorm.DB) *APIKeyService {
	return &APIKeyService{db: database}
}

func (s *APIKeyService) CreateAPIKey(userID uint, name string, expiresInDays *int) (*APIKey, error) {
	// Generate random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, err
	}
	keyString := "pb_" + hex.EncodeToString(keyBytes)

	// Calculate expiration
	var expiresAt *time.Time
	if expiresInDays != nil && *expiresInDays > 0 {
		expiry := time.Now().AddDate(0, 0, *expiresInDays)
		expiresAt = &expiry
	}

	apiKey := &APIKey{
		Key:       keyString,
		Name:      name,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}

	if err := s.db.Create(apiKey).Error; err != nil {
		return nil, err
	}

	return apiKey, nil
}

func (s *APIKeyService) ValidateAPIKey(keyString string) (*User, error) {
	var apiKey APIKey
	if err := s.db.Preload("User").Where("key = ?", keyString).First(&apiKey).Error; err != nil {
		return nil, errors.New("invalid API key")
	}

	// Check if expired
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, errors.New("API key expired")
	}

	// Update last used timestamp
	now := time.Now()
	s.db.Model(&apiKey).Update("last_used", now)

	return &apiKey.User, nil
}

func (s *APIKeyService) GetUserAPIKeys(userID uint) ([]APIKey, error) {
	var keys []APIKey
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *APIKeyService) DeleteAPIKey(keyID uint, userID uint) error {
	result := s.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&APIKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("API key not found")
	}
	return nil
}
