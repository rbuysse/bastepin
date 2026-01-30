package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
)

var authService *AuthService
var pasteService *PasteService
var apikeyService *APIKeyService
var adminService *AdminService

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := authService.Register(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create session
	session, err := authService.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"username": user.Username,
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := authService.Login(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Create session
	session, err := authService.CreateSession(user.ID)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"username": user.Username,
	})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session")
	if err == nil {
		authService.DeleteSession(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func getCurrentUser(r *http.Request) *User {
	// Check for API key in Authorization header first
	apiKey := r.Header.Get("Authorization")
	if apiKey != "" {
		// Support "Bearer <key>" or just "<key>"
		if strings.HasPrefix(apiKey, "Bearer ") {
			apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		}

		user, err := apikeyService.ValidateAPIKey(apiKey)
		if err == nil && user != nil {
			return user
		}
	}

	// Fall back to session cookie
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	session, err := authService.GetSession(cookie.Value)
	if err != nil {
		return nil
	}

	return &session.User
}

func myPastesHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pastes, err := pasteService.GetUserPastes(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch pastes", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFS(templatesFolder, "templates/my-pastes.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Username string
		Pastes   []Paste
	}{
		Username: user.Username,
		Pastes:   pastes,
	}

	tmpl.Execute(w, data)
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)

	w.Header().Set("Content-Type", "application/json")
	if user == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"username":      user.Username,
		"user_id":       user.ID,
	})
}

func editPastePageHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	pasteID := strings.TrimPrefix(r.URL.Path, "/edit/")
	if pasteID == "" {
		http.Error(w, "Invalid paste ID", http.StatusBadRequest)
		return
	}

	paste, err := pasteService.GetPaste(pasteID, &user.ID)
	if err != nil {
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}

	if paste.UserID == nil || *paste.UserID != user.ID {
		http.Error(w, "You can only edit your own pastes", http.StatusForbidden)
		return
	}

	tmpl, err := template.ParseFS(templatesFolder, "templates/edit-paste.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, paste)
}
