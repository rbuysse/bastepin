package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"unicode/utf8"
)

func notfoundHandler(w http.ResponseWriter) {
	tmpl, err := template.ParseFS(templatesFolder, "templates/404.html")
	if err != nil {
		log.Fatal(err)
	}
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
	fmt.Fprintf(w, "Upload path is %s\n", config.UploadPath)
	fmt.Fprintf(w, "%d paste hashes in memory\n", len(hashes))
}

func readyzHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "200")
}

// Serve paste
func servePasteHandler(w http.ResponseWriter, r *http.Request) {
	pasteName := filepath.Base(r.URL.Path)

	if err := validatePasteName(pasteName, config.UploadPath); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Construct the full path to the paste file.
	pastePath := filepath.Join(config.UploadPath, pasteName)

	// Open the paste file.
	pasteFile, err := os.Open(pastePath)
	if err != nil {
		// File doesn't exist - remove any stale hash entries
		if config.Debug {
			fmt.Printf("File %s not found, removing stale hash entries\n", pasteName)
		}
		removeHashForFile(pasteName)
		notfoundHandler(w)
		return
	}
	defer pasteFile.Close()

	// Set the Content-Type header for plain text.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Copy the file data to the response writer.
	_, err = io.Copy(w, pasteFile)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
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

	// Limit paste size to 10MB
	if len(body) > 10<<20 {
		http.Error(w, "Paste too large (max 10MB)", http.StatusBadRequest)
		return
	}

	// Check if it's valid UTF-8 text
	text := string(body)
	if !utf8.ValidString(text) {
		http.Error(w, "Invalid UTF-8 text", http.StatusBadRequest)
		return
	}

	writeTextAndReturnURL(w, r, text)
}
