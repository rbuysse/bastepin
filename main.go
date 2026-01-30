package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

type Config struct {
	Bind          string `toml:"bind"`
	Debug         bool   `toml:"debug"`
	ServePath     string `toml:"serve_path"`
	DatabasePath  string `toml:"database_path"`
	SessionSecret string `toml:"session_secret"`
}

var config Config

//go:embed templates
var templatesFolder embed.FS

func main() {
	config = GenerateConfig()

	// Initialize database
	if err := initDatabase(config.DatabasePath, config.Debug); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize services
	authService = NewAuthService(db)
	pasteService = NewPasteService(db)
	apikeyService = NewAPIKeyService(db)
	adminService = NewAdminService(db)

	// Clean up expired sessions and pastes periodically
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			authService.CleanupExpiredSessions()
			pasteService.CleanupExpiredPastes()
		}
	}()
	go func() {
		for {
			cleanExpiredSessions()
			time.Sleep(1 * time.Hour)
		}
	}()

	// Create a new HTTP router
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/livez", livezHandler)
	http.HandleFunc("/readyz", readyzHandler)

	// Auth endpoints
	http.HandleFunc("/api/register", registerHandler)
	http.HandleFunc("/api/login", loginHandler)
	http.HandleFunc("/api/logout", logoutHandler)
	http.HandleFunc("/api/me", meHandler)

	// Paste endpoints
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/api/paste/delete/", deletePasteHandler)
	http.HandleFunc("/api/paste/update/", updatePasteHandler)
	http.HandleFunc("/api/paste/search", searchPastesHandler)
	http.HandleFunc("/my-pastes", myPastesHandler)
	http.HandleFunc("/all", allPastesHandler)
	http.HandleFunc("/edit/", editPastePageHandler)

	// API Key endpoints
	http.HandleFunc("/api-keys", apiKeysPageHandler)
	http.HandleFunc("/api/keys/create", createAPIKeyHandler)
	http.HandleFunc("/api/keys/delete", deleteAPIKeyHandler)

	// Admin endpoints
	http.HandleFunc("/admin", adminPanelHandler)
	http.HandleFunc("/api/admin/delete-user", adminDeleteUserHandler)

	// Serve pastes
	http.HandleFunc(config.ServePath, servePasteHandler)

	// Static files and templates
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve static files
		if strings.HasPrefix(r.URL.Path, "/static/") {
			filePath := path.Join("templates", r.URL.Path)
			file, err := templatesFolder.Open(filePath)
			if err != nil {
				notfoundHandler(w)
				return
			}
			defer file.Close()

			// Set appropriate content type
			if strings.HasSuffix(r.URL.Path, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			} else if strings.HasSuffix(r.URL.Path, ".css") {
				w.Header().Set("Content-Type", "text/css")
			}

			io.Copy(w, file)
			return
		}

		// Serve index page
		if r.URL.Path == "/" {
			filePath := "templates/index.html"
			file, err := templatesFolder.Open(filePath)
			if err != nil {
				notfoundHandler(w)
				return
			}
			defer file.Close()
			io.Copy(w, file)
			return
		}

		notfoundHandler(w)
	})

	if config.Debug {
		fmt.Println("Debug mode is enabled")
	}

	fmt.Printf("Server is running on http://%s\n"+
		"Serving pastes at %s\n"+
		"Database path is %s\n",
		config.Bind, config.ServePath, config.DatabasePath)

	log.Fatal(http.ListenAndServe(config.Bind, nil))
}
