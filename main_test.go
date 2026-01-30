package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	err = db.AutoMigrate(&User{}, &Paste{}, &Session{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestAuthService_Register(t *testing.T) {
	testDB := setupTestDB(t)
	authSvc := NewAuthService(testDB)

	tests := []struct {
		name        string
		username    string
		password    string
		expectError bool
	}{
		{"Valid registration", "testuser", "password123", false},
		{"Short username", "ab", "password123", true},
		{"Short password", "testuser2", "12345", true},
		{"Duplicate username", "testuser", "password123", true},
		{"Long username", "verylongusernamethatexceedsfiftycharacterslimithere", "password123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := authSvc.Register(tt.username, tt.password)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if user.Username != tt.username {
					t.Errorf("Expected username %s, got %s", tt.username, user.Username)
				}
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	testDB := setupTestDB(t)
	authSvc := NewAuthService(testDB)

	// Create a test user
	_, err := authSvc.Register("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name        string
		username    string
		password    string
		expectError bool
	}{
		{"Valid login", "testuser", "password123", false},
		{"Wrong password", "testuser", "wrongpass", true},
		{"Non-existent user", "nouser", "password123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := authSvc.Login(tt.username, tt.password)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if user.Username != tt.username {
					t.Errorf("Expected username %s, got %s", tt.username, user.Username)
				}
			}
		})
	}
}

func TestAuthService_SessionManagement(t *testing.T) {
	testDB := setupTestDB(t)
	authSvc := NewAuthService(testDB)

	user, err := authSvc.Register("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create session
	session, err := authSvc.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.UserID != user.ID {
		t.Errorf("Expected UserID %d, got %d", user.ID, session.UserID)
	}

	// Get session
	retrievedSession, err := authSvc.GetSession(session.ID)
	if err != nil {
		t.Errorf("Failed to retrieve session: %v", err)
	}

	if retrievedSession.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, retrievedSession.ID)
	}

	// Delete session
	err = authSvc.DeleteSession(session.ID)
	if err != nil {
		t.Errorf("Failed to delete session: %v", err)
	}

	// Verify session is deleted
	_, err = authSvc.GetSession(session.ID)
	if err == nil {
		t.Errorf("Expected error when getting deleted session")
	}
}

func TestPasteService_CreatePaste(t *testing.T) {
	testDB := setupTestDB(t)
	pasteSvc := NewPasteService(testDB)
	authSvc := NewAuthService(testDB)

	user, _ := authSvc.Register("testuser", "password123")

	tests := []struct {
		name        string
		content     string
		language    string
		isPrivate   bool
		userID      *uint
		expectError bool
	}{
		{"Valid public paste", "Hello world", "text", false, nil, false},
		{"Valid private paste", "Secret", "python", true, &user.ID, false},
		{"Empty content", "", "text", false, nil, true},
		{"Private paste without user", "Secret", "text", true, nil, true},
		{"Large content", string(make([]byte, 11<<20)), "text", false, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paste, err := pasteSvc.CreatePaste("", tt.content, tt.language, tt.isPrivate, false, nil, tt.userID)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if paste.Content != tt.content {
					t.Errorf("Expected content %s, got %s", tt.content, paste.Content)
				}
				if paste.Language != tt.language {
					t.Errorf("Expected language %s, got %s", tt.language, paste.Language)
				}
			}
		})
	}
}

func TestPasteService_GetPaste(t *testing.T) {
	testDB := setupTestDB(t)
	pasteSvc := NewPasteService(testDB)
	authSvc := NewAuthService(testDB)

	user1, _ := authSvc.Register("user1", "password123")
	user2, _ := authSvc.Register("user2", "password123")

	// Create public paste
	publicPaste, _ := pasteSvc.CreatePaste("", "Public content", "text", false, false, nil, nil)

	// Create private paste
	privatePaste, _ := pasteSvc.CreatePaste("", "Private content", "text", true, false, nil, &user1.ID)

	tests := []struct {
		name        string
		pasteID     string
		viewerID    *uint
		expectError bool
	}{
		{"Public paste - anonymous", publicPaste.ID, nil, false},
		{"Public paste - logged in", publicPaste.ID, &user1.ID, false},
		{"Private paste - owner", privatePaste.ID, &user1.ID, false},
		{"Private paste - other user", privatePaste.ID, &user2.ID, true},
		{"Private paste - anonymous", privatePaste.ID, nil, true},
		{"Non-existent paste", "nonexistent", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pasteSvc.GetPaste(tt.pasteID, tt.viewerID)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPasteService_UpdatePaste(t *testing.T) {
	testDB := setupTestDB(t)
	pasteSvc := NewPasteService(testDB)
	authSvc := NewAuthService(testDB)

	user1, _ := authSvc.Register("user1", "password123")
	user2, _ := authSvc.Register("user2", "password123")

	paste, _ := pasteSvc.CreatePaste("", "Original content", "text", false, false, nil, &user1.ID)

	tests := []struct {
		name        string
		pasteID     string
		content     string
		language    string
		userID      uint
		expectError bool
	}{
		{"Valid update by owner", paste.ID, "Updated content", "python", user1.ID, false},
		{"Update by non-owner", paste.ID, "Hacked", "text", user2.ID, true},
		{"Non-existent paste", "nonexistent", "Content", "text", user1.ID, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pasteSvc.UpdatePaste(tt.pasteID, "", tt.content, tt.language, false, tt.userID)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPasteService_DeletePaste(t *testing.T) {
	testDB := setupTestDB(t)
	pasteSvc := NewPasteService(testDB)
	authSvc := NewAuthService(testDB)

	user1, _ := authSvc.Register("user1", "password123")
	user2, _ := authSvc.Register("user2", "password123")

	paste, _ := pasteSvc.CreatePaste("", "Content to delete", "text", false, false, nil, &user1.ID)

	tests := []struct {
		name        string
		pasteID     string
		userID      uint
		expectError bool
	}{
		{"Delete by owner", paste.ID, user1.ID, false},
		{"Delete by non-owner", paste.ID, user2.ID, true},
		{"Delete non-existent paste", "nonexistent", user1.ID, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pasteSvc.DeletePaste(tt.pasteID, tt.userID)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Verify paste is actually deleted
				_, err := pasteSvc.GetPaste(tt.pasteID, &tt.userID)
				if err == nil {
					t.Errorf("Expected paste to be deleted")
				}
			}
		})
	}
}

func TestPasteService_GetUserPastes(t *testing.T) {
	testDB := setupTestDB(t)
	pasteSvc := NewPasteService(testDB)
	authSvc := NewAuthService(testDB)

	user1, _ := authSvc.Register("user1", "password123")
	user2, _ := authSvc.Register("user2", "password123")

	// Create pastes for user1
	pasteSvc.CreatePaste("", "User1 paste 1", "text", false, false, nil, &user1.ID)
	pasteSvc.CreatePaste("", "User1 paste 2", "python", true, false, nil, &user1.ID)
	pasteSvc.CreatePaste("", "User1 paste 3", "javascript", false, false, nil, &user1.ID)

	// Create pastes for user2
	pasteSvc.CreatePaste("", "User2 paste 1", "text", false, false, nil, &user2.ID)

	// Get user1's pastes
	pastes, err := pasteSvc.GetUserPastes(user1.ID)
	if err != nil {
		t.Errorf("Failed to get user pastes: %v", err)
	}

	if len(pastes) != 3 {
		t.Errorf("Expected 3 pastes for user1, got %d", len(pastes))
	}

	// Get user2's pastes
	pastes, err = pasteSvc.GetUserPastes(user2.ID)
	if err != nil {
		t.Errorf("Failed to get user pastes: %v", err)
	}

	if len(pastes) != 1 {
		t.Errorf("Expected 1 paste for user2, got %d", len(pastes))
	}

	// Verify pastes are ordered by created_at DESC
	if len(pastes) > 1 {
		for i := 0; i < len(pastes)-1; i++ {
			if pastes[i].CreatedAt.Before(pastes[i+1].CreatedAt) {
				t.Errorf("Pastes not ordered by created_at DESC")
			}
		}
	}
}

func TestPasteService_CanEdit(t *testing.T) {
	testDB := setupTestDB(t)
	pasteSvc := NewPasteService(testDB)
	authSvc := NewAuthService(testDB)

	user1, _ := authSvc.Register("user1", "password123")
	user2, _ := authSvc.Register("user2", "password123")

	paste, _ := pasteSvc.CreatePaste("", "Content", "text", false, false, nil, &user1.ID)
	anonymousPaste, _ := pasteSvc.CreatePaste("", "Anonymous content", "text", false, false, nil, nil)

	tests := []struct {
		name     string
		pasteID  string
		userID   uint
		expected bool
	}{
		{"Owner can edit", paste.ID, user1.ID, true},
		{"Non-owner cannot edit", paste.ID, user2.ID, false},
		{"Cannot edit non-existent paste", "nonexistent", user1.ID, false},
		{"Cannot edit anonymous paste", anonymousPaste.ID, user1.ID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pasteSvc.CanEdit(tt.pasteID, tt.userID)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHashFunctions(t *testing.T) {
	content := "test content"
	hash1, err := computeFileHash(bytes.NewReader([]byte(content)))
	if err != nil {
		t.Errorf("Failed to compute hash: %v", err)
	}

	hash2, err := computeFileHash(bytes.NewReader([]byte(content)))
	if err != nil {
		t.Errorf("Failed to compute hash: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Expected identical hashes for same content")
	}

	hash3, err := computeFileHash(bytes.NewReader([]byte("different content")))
	if err != nil {
		t.Errorf("Failed to compute hash: %v", err)
	}

	if hash1 == hash3 {
		t.Errorf("Expected different hashes for different content")
	}
}

func TestRandFilename(t *testing.T) {
	filename1 := randfilename(8, ".txt")
	filename2 := randfilename(8, ".txt")

	if len(filename1) != 12 { // 8 chars + .txt
		t.Errorf("Expected filename length 12, got %d", len(filename1))
	}

	if filename1 == filename2 {
		t.Errorf("Expected different random filenames")
	}
}

func TestHTTPHandlers(t *testing.T) {
	// Setup
	testDB := setupTestDB(t)
	db = testDB
	authService = NewAuthService(testDB)
	pasteService = NewPasteService(testDB)
	config = Config{
		Bind:         "0.0.0.0:3001",
		ServePath:    "/p/",
		DatabasePath: ":memory:",
		Debug:        false,
	}

	t.Run("Register endpoint", func(t *testing.T) {
		reqBody := RegisterRequest{
			Username: "testuser",
			Password: "password123",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		registerHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Login endpoint", func(t *testing.T) {
		reqBody := LoginRequest{
			Username: "testuser",
			Password: "password123",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		loginHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check for session cookie
		cookies := w.Result().Cookies()
		found := false
		for _, cookie := range cookies {
			if cookie.Name == "session" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected session cookie to be set")
		}
	})

	t.Run("Upload endpoint", func(t *testing.T) {
		reqBody := UploadRequest{
			Content:   "Test paste content",
			Language:  "python",
			IsPrivate: false,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		uploadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Delete endpoint", func(t *testing.T) {
		// First register and login a user
		user, _ := authService.Register("deleteuser", "password123")
		session, _ := authService.CreateSession(user.ID)

		// Create a paste
		paste, _ := pasteService.CreatePaste("", "Content to delete", "text", false, false, nil, &user.ID)

		// Delete the paste
		req := httptest.NewRequest("POST", "/api/paste/delete/"+paste.ID, nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: session.ID})
		w := httptest.NewRecorder()

		deletePasteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Delete endpoint without auth", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/paste/delete/someid", nil)
		w := httptest.NewRecorder()

		deletePasteHandler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("My pastes endpoint", func(t *testing.T) {
		// Register and login a user
		user, _ := authService.Register("pastelistuser", "password123")
		session, _ := authService.CreateSession(user.ID)

		// Create some pastes
		pasteService.CreatePaste("", "Paste 1", "text", false, false, nil, &user.ID)
		pasteService.CreatePaste("", "Paste 2", "python", true, false, nil, &user.ID)

		req := httptest.NewRequest("GET", "/my-pastes", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: session.ID})
		w := httptest.NewRecorder()

		myPastesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("My pastes endpoint without auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/my-pastes", nil)
		w := httptest.NewRecorder()

		myPastesHandler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}
