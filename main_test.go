
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestGetSongsWithData(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	// Add some songs
	song1 := &Song{ID: "1", Name: "Song 1", CreatedAt: time.Now()}
	song2 := &Song{ID: "2", Name: "Song 2", CreatedAt: time.Now().Add(-time.Hour)}
	saveSong(song1)
	saveSong(song2)

	router := setupRouter()
	req, _ := http.NewRequest("GET", "/api/songs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var songs []*Song
	err := json.Unmarshal(w.Body.Bytes(), &songs)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(songs) != 2 {
		t.Fatalf("Expected 2 songs, got %d", len(songs))
	}

	// Songs should be returned in descending order of creation
	if songs[0].Name != "Song 1" {
		t.Errorf("Expected first song to be 'Song 1', got '%s'", songs[0].Name)
	}
	if songs[1].Name != "Song 2" {
		t.Errorf("Expected second song to be 'Song 2', got '%s'", songs[1].Name)
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

func TestRenameSongNotFound(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	router := setupRouter()

	newName := "New Name"
	payload := map[string]string{"name": newName}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", "/api/songs/non-existent-id", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestRenameSongInvalidRequest(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	// Add a song to have a valid ID
	song := &Song{
		ID:        "test-song-1",
		Name:      "Original Name",
		Original:  "uploads/test.mp3",
		Processed: "processed/test.mp3",
		CreatedAt: time.Now(),
	}
	err := saveSong(song)
	if err != nil {
		t.Fatalf("Failed to save test song: %v", err)
	}

	router := setupRouter()

	req, _ := http.NewRequest("PUT", "/api/songs/test-song-1", bytes.NewBufferString("invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteSong(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	// Add a song to delete
	song := &Song{
		ID:        "test-song-to-delete",
		Name:      "To Be Deleted",
		Original:  "uploads/to_be_deleted.mp3",
		Processed: "processed/to_be_deleted.mp3",
		CreatedAt: time.Now(),
	}
	err := saveSong(song)
	if err != nil {
		t.Fatalf("Failed to save test song for deletion: %v", err)
	}

	// Create dummy files to be deleted
	os.MkdirAll("uploads", 0755)
	os.MkdirAll("processed", 0755)
	os.Create(song.Original)
	os.Create(song.Processed)

	router := setupRouter()

	req, _ := http.NewRequest("DELETE", "/api/songs/test-song-to-delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Verify the song is deleted from the database
	_, err = getSongByID("test-song-to-delete")
	if err == nil {
		t.Error("Expected song to be deleted from DB, but it was found.")
	}

	// Verify the files are deleted
	if _, err := os.Stat(song.Original); !os.IsNotExist(err) {
		t.Errorf("Expected original file to be deleted, but it exists.")
	}
	if _, err := os.Stat(song.Processed); !os.IsNotExist(err) {
		t.Errorf("Expected processed file to be deleted, but it exists.")
	}
}

func TestDeleteSongNotFound(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	router := setupRouter()

	req, _ := http.NewRequest("DELETE", "/api/songs/non-existent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDownloadSong(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	// Create a dummy processed file in a temporary directory
	tempDir := t.TempDir()
	processedDir := filepath.Join(tempDir, "processed")
	os.MkdirAll(processedDir, 0755)
	processedPath := filepath.Join(processedDir, "test-song.mp3")
	originalPath := filepath.Join(tempDir, "uploads", "test-song.mp3")

	err := os.WriteFile(processedPath, []byte("dummy content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	song := &Song{
		ID:        "test-song",
		Name:      "Test Song",
		Original:  originalPath,
		Processed: processedPath,
		CreatedAt: time.Now(),
	}
	saveSong(song)

	router := setupRouter()
	req, _ := http.NewRequest("GET", "/api/download/test-song", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "dummy content" {
		t.Errorf("Expected file content 'dummy content', got '%s'", w.Body.String())
	}
}

func TestDownloadSongNotFound(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	router := setupRouter()
	req, _ := http.NewRequest("GET", "/api/download/non-existent-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDownloadOriginalSong(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	// Create a dummy original file in a temporary directory
	tempDir := t.TempDir()
	uploadsDir := filepath.Join(tempDir, "uploads")
	os.MkdirAll(uploadsDir, 0755)
	originalPath := filepath.Join(uploadsDir, "test-song.mp3")
	processedPath := filepath.Join(tempDir, "processed", "test-song.mp3")

	err := os.WriteFile(originalPath, []byte("original content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	song := &Song{
		ID:        "test-song",
		Name:      "Test Song",
		Original:  originalPath,
		Processed: processedPath,
		CreatedAt: time.Now(),
	}
	saveSong(song)

	router := setupRouter()
	req, _ := http.NewRequest("GET", "/api/download/test-song/original", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "original content" {
		t.Errorf("Expected file content 'original content', got '%s'", w.Body.String())
	}
}

func TestDownloadOriginalSongNotFound(t *testing.T) {
	setupTestDB(t)
	defer db.Close()

	router := setupRouter()
	req, _ := http.NewRequest("GET", "/api/download/non-existent-id/original", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetRandomUserAgent(t *testing.T) {
	agent := getRandomUserAgent()
	found := false
	for _, a := range userAgents {
		if a == agent {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("getRandomUserAgent() returned an agent not in the userAgents list: %s", agent)
	}
}

func TestCopyFile(t *testing.T) {
	srcFile, err := os.CreateTemp("", "src")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(srcFile.Name())
	srcFile.WriteString("test content")
	srcFile.Close()

	dstFile, err := os.CreateTemp("", "dst")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dstFile.Name())
	dstFile.Close()

	err = copyFile(srcFile.Name(), dstFile.Name())
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	content, err := os.ReadFile(dstFile.Name())
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got '%s'", string(content))
	}
}

func TestCopyFileSrcNotFound(t *testing.T) {
	err := copyFile("non-existent-src", "dst")
	if err == nil {
		t.Error("Expected an error for non-existent source file, but got nil")
	}
}
