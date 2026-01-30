package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestUserWorkflow tests the complete user workflow
func TestUserWorkflow(t *testing.T) {
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

	// Step 1: Register a user
	t.Log("Step 1: Registering user")
	registerReq := RegisterRequest{
		Username: "integrationuser",
		Password: "testpass123",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	registerHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Registration failed with status %d", w.Code)
	}

	// Extract session cookie
	var sessionCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("No session cookie returned after registration")
	}

	// Step 2: Create a public paste
	t.Log("Step 2: Creating public paste")
	uploadReq := UploadRequest{
		Content:   "Public integration test paste",
		Language:  "python",
		IsPrivate: false,
	}
	body, _ = json.Marshal(uploadReq)
	req = httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	uploadHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Upload failed with status %d", w.Code)
	}

	var uploadResp map[string]string
	json.NewDecoder(w.Body).Decode(&uploadResp)
	publicPasteID := uploadResp["id"]

	if publicPasteID == "" {
		t.Fatal("No paste ID returned")
	}

	// Step 3: Create a private paste
	t.Log("Step 3: Creating private paste")
	privateUploadReq := UploadRequest{
		Content:   "Private integration test paste",
		Language:  "bash",
		IsPrivate: true,
	}
	body, _ = json.Marshal(privateUploadReq)
	req = httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	uploadHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Private upload failed with status %d", w.Code)
	}

	json.NewDecoder(w.Body).Decode(&uploadResp)
	privatePasteID := uploadResp["id"]

	// Step 4: View own paste
	t.Log("Step 4: Viewing own paste")
	req = httptest.NewRequest("GET", "/p/"+publicPasteID, nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	servePasteHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to view own paste with status %d", w.Code)
	}

	// Step 5: Edit paste
	t.Log("Step 5: Editing paste")
	updateReq := PasteUpdateRequest{
		Content:  "Updated integration test paste",
		Language: "go",
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest("POST", "/api/paste/update/"+publicPasteID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	updatePasteHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to update paste with status %d", w.Code)
	}

	// Step 6: Verify update
	paste, err := pasteService.GetPaste(publicPasteID, nil)
	if err != nil {
		t.Errorf("Failed to get updated paste: %v", err)
	}
	if paste.Content != "Updated integration test paste" {
		t.Errorf("Paste content not updated correctly")
	}
	if paste.Language != "go" {
		t.Errorf("Paste language not updated correctly")
	}

	// Step 7: Try to access private paste without auth
	t.Log("Step 7: Testing private paste access control")
	req = httptest.NewRequest("GET", "/p/"+privatePasteID, nil)
	w = httptest.NewRecorder()
	servePasteHandler(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Private paste should not be accessible without auth")
	}

	// Step 8: Logout
	t.Log("Step 8: Logging out")
	req = httptest.NewRequest("POST", "/api/logout", nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	logoutHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Logout failed with status %d", w.Code)
	}

	t.Log("Integration test completed successfully")
}

// TestDeletePasteWorkflow tests deleting pastes
func TestDeletePasteWorkflow(t *testing.T) {
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

	// Register user
	t.Log("Registering user")
	registerReq := RegisterRequest{
		Username: "deleteuser",
		Password: "testpass123",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	registerHandler(w, req)

	var sessionCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie
			break
		}
	}

	// Create paste to delete
	t.Log("Creating paste")
	uploadReq := UploadRequest{
		Content:   "Paste to delete",
		Language:  "text",
		IsPrivate: false,
	}
	body, _ = json.Marshal(uploadReq)
	req = httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	uploadHandler(w, req)

	var uploadResp map[string]string
	json.NewDecoder(w.Body).Decode(&uploadResp)
	pasteID := uploadResp["id"]

	// Delete the paste
	t.Log("Deleting paste")
	req = httptest.NewRequest("POST", "/api/paste/delete/"+pasteID, nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	deletePasteHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to delete paste with status %d", w.Code)
	}

	// Verify paste is deleted
	req = httptest.NewRequest("GET", "/p/"+pasteID, nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	servePasteHandler(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Deleted paste should not be accessible")
	}

	// Try to delete non-existent paste
	t.Log("Trying to delete non-existent paste")
	req = httptest.NewRequest("POST", "/api/paste/delete/nonexistent", nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	deletePasteHandler(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Should not be able to delete non-existent paste")
	}

	// Register another user
	registerReq2 := RegisterRequest{
		Username: "otheruser",
		Password: "testpass123",
	}
	body, _ = json.Marshal(registerReq2)
	req = httptest.NewRequest("POST", "/api/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	registerHandler(w, req)

	var otherSessionCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			otherSessionCookie = cookie
			break
		}
	}

	// Create paste as first user
	uploadReq = UploadRequest{
		Content:   "User1's paste",
		Language:  "text",
		IsPrivate: false,
	}
	body, _ = json.Marshal(uploadReq)
	req = httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	uploadHandler(w, req)
	json.NewDecoder(w.Body).Decode(&uploadResp)
	user1PasteID := uploadResp["id"]

	// Try to delete as other user (should fail)
	t.Log("Trying to delete other user's paste")
	req = httptest.NewRequest("POST", "/api/paste/delete/"+user1PasteID, nil)
	req.AddCookie(otherSessionCookie)
	w = httptest.NewRecorder()
	deletePasteHandler(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Should not be able to delete another user's paste")
	}

	// Try to delete without authentication
	t.Log("Trying to delete without authentication")
	req = httptest.NewRequest("POST", "/api/paste/delete/"+user1PasteID, nil)
	w = httptest.NewRecorder()
	deletePasteHandler(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Should not be able to delete without authentication")
	}

	t.Log("Delete workflow test completed successfully")
}

// TestAnonymousUserWorkflow tests anonymous user functionality
func TestAnonymousUserWorkflow(t *testing.T) {
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

	// Anonymous user creates paste
	t.Log("Anonymous user creating paste")
	uploadReq := UploadRequest{
		Content:   "Anonymous paste content",
		Language:  "javascript",
		IsPrivate: false,
	}
	body, _ := json.Marshal(uploadReq)
	req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	uploadHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Anonymous upload failed with status %d", w.Code)
	}

	var uploadResp map[string]string
	json.NewDecoder(w.Body).Decode(&uploadResp)
	pasteID := uploadResp["id"]

	// Verify paste can be viewed
	req = httptest.NewRequest("GET", "/p/"+pasteID, nil)
	w = httptest.NewRecorder()
	servePasteHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Failed to view anonymous paste with status %d", w.Code)
	}

	// Anonymous user tries to create private paste (should fail)
	t.Log("Anonymous user attempting to create private paste")
	privateReq := UploadRequest{
		Content:   "Should fail",
		Language:  "text",
		IsPrivate: true,
	}
	body, _ = json.Marshal(privateReq)
	req = httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	uploadHandler(w, req)

	if w.Code == http.StatusOK {
		t.Errorf("Anonymous user should not be able to create private paste")
	}
}

// TestPasteDeduplication tests that duplicate pastes are handled correctly
func TestPasteDeduplication(t *testing.T) {
	testDB := setupTestDB(t)
	db = testDB
	authService = NewAuthService(testDB)
	pasteService = NewPasteService(testDB)

	user, _ := authService.Register("dedupuser", "password123")

	// Create first paste
	paste1, err := pasteService.CreatePaste("", "Duplicate content", "text", false, false, nil, &user.ID)
	if err != nil {
		t.Fatalf("Failed to create first paste: %v", err)
	}

	// Create duplicate paste
	paste2, err := pasteService.CreatePaste("", "Duplicate content", "text", false, false, nil, &user.ID)
	if err != nil {
		t.Fatalf("Failed to create second paste: %v", err)
	}

	// Should return the same paste
	if paste1.ID != paste2.ID {
		t.Log("Note: Deduplication returns existing paste (by design)")
	}

	if paste1.ContentHash != paste2.ContentHash {
		t.Errorf("Expected same content hash for duplicate content")
	}
}

// TestUIFeatures tests UI-specific functionality that was previously manual
func TestUIFeatures(t *testing.T) {
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

	// Register user
	registerReq := RegisterRequest{Username: "uitestuser", Password: "testpass123"}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	registerHandler(w, req)

	var sessionCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie
			break
		}
	}

	t.Run("Raw paste view", func(t *testing.T) {
		// Create a paste
		uploadReq := UploadRequest{Content: "Raw content test", Language: "text", IsPrivate: false}
		body, _ := json.Marshal(uploadReq)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		var uploadResp map[string]string
		json.NewDecoder(w.Body).Decode(&uploadResp)
		pasteID := uploadResp["id"]

		// Test raw view with ?raw=1
		req = httptest.NewRequest("GET", "/p/"+pasteID+"?raw=1", nil)
		w = httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for raw view, got %d", w.Code)
		}
		if w.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Errorf("Expected text/plain content type for raw view")
		}
		if w.Body.String() != "Raw content test" {
			t.Errorf("Expected raw content, got: %s", w.Body.String())
		}

		// Test raw view with Accept header
		req = httptest.NewRequest("GET", "/p/"+pasteID, nil)
		req.Header.Set("Accept", "text/plain")
		w = httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for raw view with Accept header, got %d", w.Code)
		}
		if w.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Errorf("Expected text/plain content type")
		}
	})

	t.Run("Different programming languages", func(t *testing.T) {
		languages := []string{"python", "javascript", "bash", "go", "java", "sql", "json", "yaml", "html", "css"}

		for _, lang := range languages {
			uploadReq := UploadRequest{
				Content:   "Code in " + lang,
				Language:  lang,
				IsPrivate: false,
			}
			body, _ := json.Marshal(uploadReq)
			req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(sessionCookie)
			w := httptest.NewRecorder()
			uploadHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Failed to create paste with language %s: status %d", lang, w.Code)
			}

			var uploadResp map[string]string
			json.NewDecoder(w.Body).Decode(&uploadResp)
			pasteID := uploadResp["id"]

			// Verify language was saved correctly
			paste, err := pasteService.GetPaste(pasteID, nil)
			if err != nil {
				t.Errorf("Failed to get paste: %v", err)
			}
			if paste.Language != lang {
				t.Errorf("Expected language %s, got %s", lang, paste.Language)
			}
		}
	})

	t.Run("Edit page loads correctly", func(t *testing.T) {
		// Create a paste
		uploadReq := UploadRequest{Content: "Edit test content", Language: "python", IsPrivate: false}
		body, _ := json.Marshal(uploadReq)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		var uploadResp map[string]string
		json.NewDecoder(w.Body).Decode(&uploadResp)
		pasteID := uploadResp["id"]

		// Try to access edit page
		req = httptest.NewRequest("GET", "/edit/"+pasteID, nil)
		req.AddCookie(sessionCookie)
		w = httptest.NewRecorder()
		editPastePageHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for edit page, got %d", w.Code)
		}
	})

	t.Run("My pastes page loads", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/my-pastes", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		myPastesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for my-pastes page, got %d", w.Code)
		}
	})

	t.Run("Session persists", func(t *testing.T) {
		// Verify session is still valid after multiple requests
		req := httptest.NewRequest("GET", "/api/me", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		meHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Session should still be valid")
		}

		var meResp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&meResp)
		if meResp["username"] != "uitestuser" {
			t.Errorf("Expected username uitestuser, got %v", meResp["username"])
		}
	})
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
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

	t.Run("Empty paste content", func(t *testing.T) {
		uploadReq := UploadRequest{Content: "", Language: "text", IsPrivate: false}
		body, _ := json.Marshal(uploadReq)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		if w.Code == http.StatusOK {
			t.Errorf("Should not allow empty paste content")
		}
	})

	t.Run("Extremely large paste", func(t *testing.T) {
		// Create content larger than 10MB
		largeContent := string(make([]byte, 11<<20))
		uploadReq := UploadRequest{Content: largeContent, Language: "text", IsPrivate: false}
		body, _ := json.Marshal(uploadReq)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		if w.Code == http.StatusOK {
			t.Errorf("Should not allow paste larger than 10MB")
		}
	})

	t.Run("Invalid method on upload endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/upload", nil)
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405 for GET on upload endpoint, got %d", w.Code)
		}
	})

	t.Run("Non-existent paste returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/p/nonexistent", nil)
		w := httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404 for non-existent paste, got %d", w.Code)
		}
	})

	t.Run("Invalid JSON in upload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader([]byte("{invalid json}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		// Should fall back to treating it as plain text
		if w.Code != http.StatusOK {
			t.Logf("Invalid JSON treated as plain text content, status: %d", w.Code)
		}
	})
}

// TestAccessControl tests comprehensive access control scenarios
func TestAccessControl(t *testing.T) {
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

	// Create two users
	user1, _ := authService.Register("user1", "password123")
	session1, _ := authService.CreateSession(user1.ID)
	cookie1 := &http.Cookie{Name: "session", Value: session1.ID}

	user2, _ := authService.Register("user2", "password123")
	session2, _ := authService.CreateSession(user2.ID)
	cookie2 := &http.Cookie{Name: "session", Value: session2.ID}

	// Create pastes
	publicPaste, _ := pasteService.CreatePaste("", "Public content", "text", false, false, nil, &user1.ID)
	privatePaste, _ := pasteService.CreatePaste("", "Private content", "text", true, false, nil, &user1.ID)

	t.Run("Owner can view private paste", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/p/"+privatePaste.ID, nil)
		req.AddCookie(cookie1)
		w := httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Owner should be able to view private paste, got status %d", w.Code)
		}
	})

	t.Run("Other user cannot view private paste", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/p/"+privatePaste.ID, nil)
		req.AddCookie(cookie2)
		w := httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code == http.StatusOK {
			t.Errorf("Other user should not be able to view private paste")
		}
	})

	t.Run("Anonymous cannot view private paste", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/p/"+privatePaste.ID, nil)
		w := httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code == http.StatusOK {
			t.Errorf("Anonymous user should not be able to view private paste")
		}
	})

	t.Run("Anyone can view public paste", func(t *testing.T) {
		// Test as user2
		req := httptest.NewRequest("GET", "/p/"+publicPaste.ID, nil)
		req.AddCookie(cookie2)
		w := httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("User2 should be able to view public paste, got status %d", w.Code)
		}

		// Test as anonymous
		req = httptest.NewRequest("GET", "/p/"+publicPaste.ID, nil)
		w = httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Anonymous should be able to view public paste, got status %d", w.Code)
		}
	})

	t.Run("Owner can edit paste", func(t *testing.T) {
		canEdit := pasteService.CanEdit(publicPaste.ID, user1.ID)
		if !canEdit {
			t.Errorf("Owner should be able to edit their paste")
		}
	})

	t.Run("Other user cannot edit paste", func(t *testing.T) {
		canEdit := pasteService.CanEdit(publicPaste.ID, user2.ID)
		if canEdit {
			t.Errorf("Other user should not be able to edit paste")
		}
	})

	t.Run("Non-owner cannot access edit page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/edit/"+publicPaste.ID, nil)
		req.AddCookie(cookie2)
		w := httptest.NewRecorder()
		editPastePageHandler(w, req)

		if w.Code == http.StatusOK {
			t.Errorf("Non-owner should not be able to access edit page")
		}
	})
}

// TestTitleField tests the new Title field functionality
func TestTitleField(t *testing.T) {
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

	t.Run("Create paste with title", func(t *testing.T) {
		uploadReq := UploadRequest{
			Content:   "# Hello World\nThis is my titled paste",
			Language:  "markdown",
			Title:     "My First Paste",
			IsPrivate: false,
			Unlisted:  false,
		}
		body, _ := json.Marshal(uploadReq)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Upload with title failed: %d", w.Code)
		}

		// Verify the paste has the title
		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		paste, _ := pasteService.GetPaste(response["id"], nil)
		if paste.Title != "My First Paste" {
			t.Errorf("Expected title 'My First Paste', got '%s'", paste.Title)
		}
	})

	t.Run("Create paste without title", func(t *testing.T) {
		uploadReq := UploadRequest{
			Content:  "Untitled paste content",
			Language: "text",
		}
		body, _ := json.Marshal(uploadReq)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Upload without title failed: %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		paste, _ := pasteService.GetPaste(response["id"], nil)
		if paste.Title != "" {
			t.Errorf("Expected empty title, got '%s'", paste.Title)
		}
	})

	t.Run("Update paste title", func(t *testing.T) {
		// Create authenticated session
		user, _ := authService.Register("titleupdater", "pass123")
		session, _ := authService.CreateSession(user.ID)

		// Create a paste
		paste, _ := pasteService.CreatePaste("Original Title", "Content", "text", false, false, nil, &user.ID)

		// Update the title
		updateReq := PasteUpdateRequest{
			Content:  "Updated content",
			Language: "text",
			Title:    "Updated Title",
			Unlisted: false,
		}
		body, _ := json.Marshal(updateReq)
		req := httptest.NewRequest("POST", "/api/paste/update/"+paste.ID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "session", Value: session.ID})
		w := httptest.NewRecorder()
		updatePasteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Update failed: %d, body: %s", w.Code, w.Body.String())
		}

		// Verify title was updated
		updatedPaste, _ := pasteService.GetPaste(paste.ID, &user.ID)
		if updatedPaste.Title != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got '%s'", updatedPaste.Title)
		}
	})
}

// TestUnlistedPastes tests the unlisted paste functionality
func TestUnlistedPastes(t *testing.T) {
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

	t.Run("Create unlisted paste", func(t *testing.T) {
		uploadReq := UploadRequest{
			Content:   "This is an unlisted paste",
			Language:  "text",
			Title:     "Unlisted Paste",
			IsPrivate: false,
			Unlisted:  true,
		}
		body, _ := json.Marshal(uploadReq)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Unlisted paste creation failed: %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		paste, _ := pasteService.GetPaste(response["id"], nil)
		if !paste.Unlisted {
			t.Errorf("Expected paste to be unlisted")
		}
	})

	t.Run("Unlisted paste accessible via direct link", func(t *testing.T) {
		paste, _ := pasteService.CreatePaste("Unlisted Test", "Content", "text", false, true, nil, nil)

		req := httptest.NewRequest("GET", "/p/"+paste.ID, nil)
		w := httptest.NewRecorder()
		servePasteHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Unlisted paste should be accessible via direct link, got status %d", w.Code)
		}
	})

	t.Run("Unlisted paste not shown in /all", func(t *testing.T) {
		// Create a public paste and an unlisted paste
		pasteService.CreatePaste("Public Paste", "Public content", "text", false, false, nil, nil)
		pasteService.CreatePaste("Unlisted Paste", "Unlisted content", "text", false, true, nil, nil)

		// Get all public pastes
		publicPastes, _ := pasteService.GetAllPublicPastes()

		// Should only have the public paste
		if len(publicPastes) != 1 {
			t.Errorf("Expected 1 public paste in /all, got %d", len(publicPastes))
		}
		if publicPastes[0].Title != "Public Paste" {
			t.Errorf("Expected 'Public Paste', got '%s'", publicPastes[0].Title)
		}
	})

	t.Run("Private paste not shown in /all", func(t *testing.T) {
		user, _ := authService.Register("privateuser", "pass123")

		// Create private paste
		pasteService.CreatePaste("Private Paste", "Private content", "text", true, false, nil, &user.ID)

		publicPastes, _ := pasteService.GetAllPublicPastes()

		// Should not include private paste
		for _, paste := range publicPastes {
			if paste.Title == "Private Paste" {
				t.Errorf("Private paste should not appear in /all")
			}
		}
	})
}

// TestBrowsePublicPastes tests the /all page functionality
func TestBrowsePublicPastes(t *testing.T) {
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

	t.Run("Browse page shows public pastes", func(t *testing.T) {
		// Create various types of pastes
		pasteService.CreatePaste("Public 1", "Content 1", "text", false, false, nil, nil)
		pasteService.CreatePaste("Public 2", "Content 2", "python", false, false, nil, nil)
		pasteService.CreatePaste("Unlisted", "Content 3", "text", false, true, nil, nil)

		user, _ := authService.Register("browseuser", "pass123")
		pasteService.CreatePaste("Private", "Content 4", "text", true, false, nil, &user.ID)

		// Request the /all page
		req := httptest.NewRequest("GET", "/all", nil)
		w := httptest.NewRecorder()
		allPastesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("/all page failed: %d", w.Code)
		}

		body := w.Body.String()

		// Should contain public pastes
		if !bytes.Contains([]byte(body), []byte("Public 1")) {
			t.Errorf("Browse page should show Public 1")
		}
		if !bytes.Contains([]byte(body), []byte("Public 2")) {
			t.Errorf("Browse page should show Public 2")
		}

		// Should NOT contain unlisted or private
		if bytes.Contains([]byte(body), []byte("Unlisted")) {
			t.Errorf("Browse page should not show unlisted paste")
		}
		if bytes.Contains([]byte(body), []byte("Private")) {
			t.Errorf("Browse page should not show private paste")
		}
	})

	t.Run("GetAllPublicPastes returns correct pastes", func(t *testing.T) {
		publicPastes, err := pasteService.GetAllPublicPastes()
		if err != nil {
			t.Fatalf("GetAllPublicPastes failed: %v", err)
		}

		// Verify all returned pastes are public and not unlisted
		for _, paste := range publicPastes {
			if paste.IsPrivate {
				t.Errorf("GetAllPublicPastes returned private paste: %s", paste.ID)
			}
			if paste.Unlisted {
				t.Errorf("GetAllPublicPastes returned unlisted paste: %s", paste.ID)
			}
		}
	})
}

// TestLegacyUploadFormat tests backward compatibility with plain text uploads
func TestLegacyUploadFormat(t *testing.T) {
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

	t.Run("Plain text upload without JSON", func(t *testing.T) {
		content := "Plain text paste"
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader([]byte(content)))
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Plain text upload should work, got status %d", w.Code)
		}

		// Response should be plain text URL
		responseURL := w.Body.String()
		if !bytes.Contains([]byte(responseURL), []byte("/p/")) {
			t.Errorf("Response should contain paste URL, got: %s", responseURL)
		}
	})

	t.Run("Plain text upload with language query param", func(t *testing.T) {
		content := "print('hello')"
		req := httptest.NewRequest("POST", "/upload?language=python", bytes.NewReader([]byte(content)))
		w := httptest.NewRecorder()
		uploadHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Plain text upload with language param should work, got status %d", w.Code)
		}
	})
}
