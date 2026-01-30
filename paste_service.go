package main

import (
	"bytes"
	"errors"
	"time"

	"gorm.io/gorm"
)

type PasteService struct {
	db *gorm.DB
}

func NewPasteService(database *gorm.DB) *PasteService {
	return &PasteService{db: database}
}

func (s *PasteService) CreatePaste(title, content, language string, isPrivate, unlisted bool, expiresIn *int, userID *uint) (*Paste, error) {
	if len(content) == 0 {
		return nil, errors.New("paste content cannot be empty")
	}

	if len(content) > 10<<20 { // 10MB
		return nil, errors.New("paste too large (max 10MB)")
	}

	// Anonymous users cannot create private pastes
	if isPrivate && userID == nil {
		return nil, errors.New("must be logged in to create private pastes")
	}

	// Calculate expiration time
	var expiresAt *time.Time
	if expiresIn != nil && *expiresIn > 0 {
		expiry := time.Now().Add(time.Duration(*expiresIn) * time.Minute)
		expiresAt = &expiry
	}

	// Compute hash for deduplication
	hash, err := computeFileHash(bytes.NewReader([]byte(content)))
	if err != nil {
		return nil, err
	}

	// Check if identical paste exists for this user (or public if anonymous)
	var existingPaste Paste
	query := s.db.Where("content_hash = ?", hash)
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	if err := query.First(&existingPaste).Error; err == nil {
		// Identical paste exists, return it
		return &existingPaste, nil
	}

	// Generate unique ID
	pasteID := randfilename(8, "")

	paste := &Paste{
		ID:          pasteID,
		Title:       title,
		Content:     content,
		ContentHash: hash,
		Language:    language,
		IsPrivate:   isPrivate,
		Unlisted:    unlisted,
		ExpiresAt:   expiresAt,
		UserID:      userID,
	}

	if err := s.db.Create(paste).Error; err != nil {
		return nil, err
	}

	return paste, nil
}

func (s *PasteService) GetPaste(pasteID string, viewerUserID *uint) (*Paste, error) {
	var paste Paste
	if err := s.db.Preload("User").Where("id = ?", pasteID).First(&paste).Error; err != nil {
		return nil, errors.New("paste not found")
	}

	// Check if paste has expired
	if paste.ExpiresAt != nil && time.Now().After(*paste.ExpiresAt) {
		return nil, errors.New("paste not found")
	}

	// Check privacy
	if paste.IsPrivate {
		// Only owner can view private pastes
		if viewerUserID == nil || paste.UserID == nil || *viewerUserID != *paste.UserID {
			return nil, errors.New("paste not found")
		}
	}

	return &paste, nil
}

func (s *PasteService) UpdatePaste(pasteID, title, content, language string, unlisted bool, userID uint) (*Paste, error) {
	var paste Paste
	if err := s.db.Where("id = ?", pasteID).First(&paste).Error; err != nil {
		return nil, errors.New("paste not found")
	}

	// Check ownership
	if paste.UserID == nil || *paste.UserID != userID {
		return nil, errors.New("you can only edit your own pastes")
	}

	// Update content and hash
	hash, err := computeFileHash(bytes.NewReader([]byte(content)))
	if err != nil {
		return nil, err
	}

	paste.Title = title
	paste.Content = content
	paste.ContentHash = hash
	paste.Language = language
	paste.Unlisted = unlisted
	paste.UpdatedAt = time.Now()

	if err := s.db.Save(&paste).Error; err != nil {
		return nil, err
	}

	return &paste, nil
}

func (s *PasteService) GetUserPastes(userID uint) ([]Paste, error) {
	var pastes []Paste
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&pastes).Error; err != nil {
		return nil, err
	}
	return pastes, nil
}

func (s *PasteService) CanEdit(pasteID string, userID uint) bool {
	var paste Paste
	if err := s.db.Where("id = ? AND user_id = ?", pasteID, userID).First(&paste).Error; err != nil {
		return false
	}
	return true
}

func (s *PasteService) DeletePaste(pasteID string, userID uint) error {
	var paste Paste
	if err := s.db.Where("id = ?", pasteID).First(&paste).Error; err != nil {
		return errors.New("paste not found")
	}

	// Check ownership
	if paste.UserID == nil || *paste.UserID != userID {
		return errors.New("you can only delete your own pastes")
	}

	if err := s.db.Delete(&paste).Error; err != nil {
		return err
	}

	return nil
}

func (s *PasteService) GetAllPublicPastes() ([]Paste, error) {
	var pastes []Paste
	if err := s.db.Preload("User").Where("is_private = ? AND unlisted = ?", false, false).Order("created_at DESC").Find(&pastes).Error; err != nil {
		return nil, err
	}
	return pastes, nil
}

func (s *PasteService) SearchUserPastes(userID uint, query string) ([]Paste, error) {
	var pastes []Paste
	searchPattern := "%" + query + "%"
	if err := s.db.Where("user_id = ? AND (title LIKE ? OR content LIKE ?)", userID, searchPattern, searchPattern).
		Order("created_at DESC").
		Find(&pastes).Error; err != nil {
		return nil, err
	}
	return pastes, nil
}

func (s *PasteService) CleanupExpiredPastes() (int64, error) {
	now := time.Now()
	result := s.db.Where("expires_at IS NOT NULL AND expires_at < ?", now).Delete(&Paste{})
	return result.RowsAffected, result.Error
}
