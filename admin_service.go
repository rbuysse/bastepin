package main

import (
	"errors"

	"gorm.io/gorm"
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(database *gorm.DB) *AdminService {
	return &AdminService{db: database}
}

func (s *AdminService) IsAdmin(userID uint) bool {
	var admin Admin
	err := s.db.Where("user_id = ?", userID).First(&admin).Error
	return err == nil
}

func (s *AdminService) MakeAdmin(userID uint) error {
	admin := &Admin{UserID: userID}
	return s.db.Create(admin).Error
}

func (s *AdminService) RemoveAdmin(userID uint) error {
	result := s.db.Where("user_id = ?", userID).Delete(&Admin{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user is not an admin")
	}
	return nil
}

func (s *AdminService) GetAllUsers() ([]User, error) {
	var users []User
	if err := s.db.Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AdminService) GetUserStats(userID uint) (map[string]interface{}, error) {
	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	var pasteCount int64
	s.db.Model(&Paste{}).Where("user_id = ?", userID).Count(&pasteCount)

	var sessionCount int64
	s.db.Model(&Session{}).Where("user_id = ?", userID).Count(&sessionCount)

	return map[string]interface{}{
		"username":      user.Username,
		"created_at":    user.CreatedAt,
		"paste_count":   pasteCount,
		"session_count": sessionCount,
	}, nil
}

func (s *AdminService) DeleteUser(userID uint) error {
	// Delete user's sessions
	s.db.Where("user_id = ?", userID).Delete(&Session{})

	// Delete user's API keys
	s.db.Where("user_id = ?", userID).Delete(&APIKey{})

	// Delete user's pastes
	s.db.Where("user_id = ?", userID).Delete(&Paste{})

	// Remove admin status if exists
	s.db.Where("user_id = ?", userID).Delete(&Admin{})

	// Delete user
	result := s.db.Delete(&User{}, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}
