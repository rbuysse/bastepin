package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Authentication service
type AuthService struct {
	db *gorm.DB
}

func NewAuthService(database *gorm.DB) *AuthService {
	return &AuthService{db: database}
}

func (s *AuthService) Register(username, password string) (*User, error) {
	if len(username) < 3 || len(username) > 50 {
		return nil, errors.New("username must be between 3 and 50 characters")
	}
	
	if len(password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		Username:     username,
		PasswordHash: string(hashedPassword),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, errors.New("username already exists")
	}

	return user, nil
}

func (s *AuthService) Login(username, password string) (*User, error) {
	var user User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, errors.New("invalid username or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid username or password")
	}

	return &user, nil
}

func (s *AuthService) CreateSession(userID uint) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
	}

	if err := s.db.Create(session).Error; err != nil {
		return nil, err
	}

	return session, nil
}

func (s *AuthService) GetSession(sessionID string) (*Session, error) {
	var session Session
	if err := s.db.Preload("User").Where("id = ? AND expires_at > ?", sessionID, time.Now()).First(&session).Error; err != nil {
		return nil, errors.New("invalid or expired session")
	}

	return &session, nil
}

func (s *AuthService) DeleteSession(sessionID string) error {
	return s.db.Where("id = ?", sessionID).Delete(&Session{}).Error
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (s *AuthService) CleanupExpiredSessions() (int64, error) {
	result := s.db.Where("expires_at < ?", time.Now()).Delete(&Session{})
	return result.RowsAffected, result.Error
}
