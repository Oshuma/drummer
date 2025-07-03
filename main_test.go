
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// Setup a test router
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	api := r.Group("/api")
	{
		api.POST("/upload", uploadSong)
		api.POST("/youtube", downloadYoutube)
		api.GET("/songs", getSongs)
		api.GET("/download/:id", downloadSong)
		api.GET("/download/:id/original", downloadOriginalSong)
		api.DELETE("/songs/:id", deleteSong)
		api.PUT("/songs/:id", renameSong)
		api.GET("/version", getVersion)
	}
	return r
}

// Setup a temporary database for testing
func setupTestDB(t *testing.T) {
	// Create a temporary directory for the database
	tempDir := t.TempDir()
	dbPath := tempDir + "/test_songs.db"
	
	// Set up the database connection for the tests
	os.Setenv("DB_PATH", dbPath)
	initDB()
}

func TestMain(m *testing.M) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Run tests
	exitVal := m.Run()

	// Clean up
	os.Remove("./data/test_songs.db")

	os.Exit(exitVal)
}

func TestGetVersion(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["name"] != "Drummer" {
		t.Errorf("Expected name 'Drummer', got '%s'", response["name"])
	}

	if response["version"] != Version {
		t.Errorf("Expected version '%s', got '%s'", Version, response["version"])
	}
}

func TestGetSongsEmpty(t *testing.T) {
	// Setup a clean database for this test
	setupTestDB(t)
	defer db.Close()

	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/songs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// The body should be an empty JSON array
	expected := "[]"
	if w.Body.String() != expected {
		t.Errorf("Expected empty array, got '%s'", w.Body.String())
	}
}

func TestRenameSong(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	// First, add a song to the database to test renaming
	song := &Song{
		ID:        "test-song-1",
		Name:      "Original Name",
		Original:  "uploads/test.mp3",
		Processed: "processed/test.mp3",
		CreatedAt:   time.Now(),
	}
	err := saveSong(song)
	if err != nil {
		t.Fatalf("Failed to save test song: %v", err)
	}

	router := setupRouter()

	// New name for the song
	newName := "New Awesome Name"
	payload := map[string]string{"name": newName}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", "/api/songs/test-song-1", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Verify the name was updated in the database
	updatedSong, err := getSongByID("test-song-1")
	if err != nil {
		t.Fatalf("Failed to retrieve updated song: %v", err)
	}

	if updatedSong.Name != newName {
		t.Errorf("Expected song name to be '%s', but got '%s'", newName, updatedSong.Name)
	}
}
