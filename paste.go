package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func randfilename(length int, extension string) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomRunes := make([]rune, length)
	seed := rand.NewSource(time.Now().UnixNano())
	rand := rand.New(seed)
	for index := range randomRunes {
		randomRunes[index] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(randomRunes) + extension
}

func writeTextAndReturnURL(w http.ResponseWriter, r *http.Request, text string) error {
	// Compute hash from text
	hash, err := computeFileHash(bytes.NewReader([]byte(text)))
	if err != nil {
		return err
	}
	value, exists := pasteHashExists(hash)

	if exists {
		// Verify the file actually exists on disk
		filePath := filepath.Join(config.UploadPath, value)
		if _, err := os.Stat(filePath); err == nil {
			// File exists, return existing URL
			if config.Debug {
				fmt.Printf("Hash %s exists: %s\n", hash, value)
			}
			serveURL := fmt.Sprintf("%s%s", config.ServePath, value)
			fmt.Fprintf(w, serveURL)
			return nil
		} else {
			// Hash exists but file doesn't - remove stale hash
			if config.Debug {
				fmt.Printf("Removing stale hash %s for missing file %s\n", hash, value)
			}
			removeHashForFile(value)
		}
	}

	// Create new paste file
	filename := randfilename(8, ".txt")
	filePath := filepath.Join(config.UploadPath, filename)

	// Check if file already exists (extremely unlikely but possible)
	for {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			break
		}
		filename = randfilename(8, ".txt")
		filePath = filepath.Join(config.UploadPath, filename)
	}

	// Write text to file
	err = os.WriteFile(filePath, []byte(text), 0644)
	if err != nil {
		return err
	}

	// Add hash to dictionary
	addHashForFile(hash, filename)

	// Return the URL
	serveURL := fmt.Sprintf("%s%s", config.ServePath, filename)
	fmt.Fprintf(w, serveURL)

	if config.Debug {
		fmt.Printf("New paste: %s (hash: %s)\n", filename, hash)
	}

	return nil
}

func validatePasteName(filename string, uploadPath string) error {
	// Check for path traversal attempts
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return fmt.Errorf("invalid filename")
	}

	// Check if file has .txt extension
	if !strings.HasSuffix(filename, ".txt") {
		return fmt.Errorf("invalid file extension")
	}

	return nil
}
