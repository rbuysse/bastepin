package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"
)

type UploadRequest struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	Language  string `json:"language"`
	IsPrivate bool   `json:"is_private"`
	Unlisted  bool   `json:"unlisted"`
	ExpiresIn *int   `json:"expires_in"` // minutes until expiration, nil = never
}

type PasteUpdateRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Unlisted bool   `json:"unlisted"`
}

func notfoundHandler(w http.ResponseWriter) {
	tmpl, err := template.ParseFS(templatesFolder, "templates/404.html")
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(http.StatusNotFound)
	tmpl.Execute(w, nil)
}

func livezHandler(w http.ResponseWriter, req *http.Request) {
	_, verbose := req.URL.Query()["verbose"]
	if !verbose {
		fmt.Fprintf(w, "200")
		return
	}
	// Print extra info if verbose is present http://foo.bar:3001/livez?verbose
	fmt.Fprintf(w, "Server is running on http://%s\n", config.Bind)
	fmt.Fprintf(w, "Serving pastes at %s\n", config.ServePath)
	fmt.Fprintf(w, "Database path is %s\n", config.DatabasePath)
}

func readyzHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "200")
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
}

// Serve paste
func servePasteHandler(w http.ResponseWriter, r *http.Request) {
	pasteID := strings.TrimPrefix(r.URL.Path, config.ServePath)

	if pasteID == "" {
		notfoundHandler(w)
		return
	}

	// Get current user (may be nil for anonymous)
	user := getCurrentUser(r)
	var userID *uint
	if user != nil {
		userID = &user.ID
	}

	paste, err := pasteService.GetPaste(pasteID, userID)
	if err != nil {
		notfoundHandler(w)
		return
	}

	// Check if this is an API request (raw paste)
	if r.URL.Query().Get("raw") == "1" || r.Header.Get("Accept") == "text/plain" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, paste.Content)
		return
	}

	// Render HTML view with syntax highlighting
	tmpl, err := template.ParseFS(templatesFolder, "templates/view-paste.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Paste    *Paste
		CanEdit  bool
		Username string
	}{
		Paste:   paste,
		CanEdit: user != nil && paste.UserID != nil && *paste.UserID == user.ID,
	}

	if user != nil {
		data.Username = user.Username
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the raw text from the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading paste", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Check if body is empty
	if len(body) == 0 {
		http.Error(w, "Empty paste", http.StatusBadRequest)
		return
	}

	// Check if it's valid UTF-8 text
	text := string(body)
	if !utf8.ValidString(text) {
		http.Error(w, "Invalid UTF-8 text", http.StatusBadRequest)
		return
	}

	// Default values
	title := ""
	language := "text"
	isPrivate := false
	unlisted := false
	var expiresIn *int

	// Try to parse as JSON for new API
	var uploadReq UploadRequest
	if err := json.Unmarshal(body, &uploadReq); err == nil {
		// JSON was parsed successfully
		if uploadReq.Content == "" {
			http.Error(w, "Empty paste", http.StatusBadRequest)
			return
		}
		title = uploadReq.Title
		text = uploadReq.Content
		if uploadReq.Language != "" {
			language = uploadReq.Language
		}
		isPrivate = uploadReq.IsPrivate
		unlisted = uploadReq.Unlisted
		expiresIn = uploadReq.ExpiresIn
	} else {
		// Legacy plain text upload - check query params
		language = r.URL.Query().Get("language")
		if language == "" {
			language = "text"
		}
		isPrivate = r.URL.Query().Get("private") == "1"
		unlisted = r.URL.Query().Get("unlisted") == "1"
	}

	// Get current user
	user := getCurrentUser(r)
	var userID *uint
	if user != nil {
		userID = &user.ID
	}

	// Anonymous users cannot create private pastes
	if isPrivate && userID == nil {
		http.Error(w, "Must be logged in to create private pastes", http.StatusUnauthorized)
		return
	}

	paste, err := pasteService.CreatePaste(title, text, language, isPrivate, unlisted, expiresIn, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serveURL := fmt.Sprintf("%s%s", config.ServePath, paste.ID)

	// Return JSON if request was JSON, otherwise plain text
	if r.Header.Get("Content-Type") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"url": serveURL,
			"id":  paste.ID,
		})
	} else {
		fmt.Fprintf(w, serveURL)
	}

	if config.Debug {
		fmt.Printf("New paste: %s (user: %v, private: %v, language: %s)\n", paste.ID, userID, isPrivate, language)
	}
}

func updatePasteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pasteID := strings.TrimPrefix(r.URL.Path, "/api/paste/update/")

	var req PasteUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	paste, err := pasteService.UpdatePaste(pasteID, req.Title, req.Content, req.Language, req.Unlisted, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"paste":   paste,
	})
}

func deletePasteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pasteID := strings.TrimPrefix(r.URL.Path, "/api/paste/delete/")

	if err := pasteService.DeletePaste(pasteID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"success": true,
	})
}

func allPastesHandler(w http.ResponseWriter, r *http.Request) {
	pastes, err := pasteService.GetAllPublicPastes()
	if err != nil {
		http.Error(w, "Error loading pastes", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFS(templatesFolder, "templates/all-pastes.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	user := getCurrentUser(r)
	data := struct {
		Pastes   []Paste
		Username string
	}{
		Pastes: pastes,
	}
	if user != nil {
		data.Username = user.Username
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

// API Key handlers
func apiKeysPageHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	keys, _ := apikeyService.GetUserAPIKeys(user.ID)

	tmpl, err := template.ParseFS(templatesFolder, "templates/api-keys.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Username string
		Keys     []APIKey
	}{
		Username: user.Username,
		Keys:     keys,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func createAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name          string `json:"name"`
		ExpiresInDays *int   `json:"expires_in_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	apiKey, err := apikeyService.CreateAPIKey(user.ID, req.Name, req.ExpiresInDays)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiKey)
}

func deleteAPIKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ID uint `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := apikeyService.DeleteAPIKey(req.ID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Search handler
func searchPastesHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	pastes, err := pasteService.SearchUserPastes(user.ID, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pastes)
}

// Admin handlers
func adminPanelHandler(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil || !adminService.IsAdmin(user.ID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	users, _ := adminService.GetAllUsers()

	tmpl, err := template.ParseFS(templatesFolder, "templates/admin-panel.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Username string
		Users    []User
	}{
		Username: user.Username,
		Users:    users,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func adminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getCurrentUser(r)
	if user == nil || !adminService.IsAdmin(user.ID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		UserID uint `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Can't delete yourself
	if req.UserID == user.ID {
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	if err := adminService.DeleteUser(req.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
