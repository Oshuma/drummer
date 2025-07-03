package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const Version = "0.1.0"

type Song struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Original  string    `json:"original"`
	Processed string    `json:"processed"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	// Initialize database
	initDB()
	defer db.Close()

	// Clean up any leftover temporary files on startup
	cleanupTempFiles()

	r := gin.Default()

	// Serve static files
	r.Static("/static", "./web/build/static")
	r.StaticFile("/", "./web/build/index.html")
	r.StaticFile("/favicon.ico", "./web/build/favicon.ico")

	// Create uploads directory
	os.MkdirAll("uploads", 0755)
	os.MkdirAll("processed", 0755)
	os.MkdirAll("temp", 0755)

	// API routes
	api := r.Group("/api")
	{
		api.POST("/upload", uploadSong)
		api.GET("/songs", getSongs)
		api.GET("/download/:id", downloadSong)
		api.GET("/download/:id/original", downloadOriginalSong)
		api.DELETE("/songs/:id", deleteSong)
		api.PUT("/songs/:id", renameSong)
		api.GET("/version", getVersion)
	}

	r.Run(":8080")
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./data/songs.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Create data directory
	os.MkdirAll("data", 0755)

	// Create songs table
	createTable := `
	CREATE TABLE IF NOT EXISTS songs (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		original_path TEXT NOT NULL,
		processed_path TEXT NOT NULL,
		created_at DATETIME NOT NULL
	);`

	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

func saveSong(song *Song) error {
	query := `INSERT INTO songs (id, name, original_path, processed_path, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, song.ID, song.Name, song.Original, song.Processed, song.CreatedAt)
	return err
}

func getSongByID(id string) (*Song, error) {
	query := `SELECT id, name, original_path, processed_path, created_at FROM songs WHERE id = ?`
	row := db.QueryRow(query, id)
	
	var song Song
	err := row.Scan(&song.ID, &song.Name, &song.Original, &song.Processed, &song.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &song, nil
}

func getAllSongs() ([]*Song, error) {
	query := `SELECT id, name, original_path, processed_path, created_at FROM songs ORDER BY created_at DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []*Song
	for rows.Next() {
		var song Song
		err := rows.Scan(&song.ID, &song.Name, &song.Original, &song.Processed, &song.CreatedAt)
		if err != nil {
			return nil, err
		}
		songs = append(songs, &song)
	}
	return songs, nil
}

func deleteSongFromDB(id string) error {
	query := `DELETE FROM songs WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

func updateSongName(id, name string) error {
	query := `UPDATE songs SET name = ? WHERE id = ?`
	_, err := db.Exec(query, name, id)
	return err
}

func uploadSong(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".mp3") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only MP3 files are supported"})
		return
	}

	// Generate unique ID
	id := uuid.New().String()
	originalPath := filepath.Join("uploads", id+".mp3")
	processedPath := filepath.Join("processed", id+".mp3")

	// Save uploaded file
	dst, err := os.Create(originalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		// Clean up partial file if copy fails
		os.Remove(originalPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Process the file to remove drums
	err = removeDrums(originalPath, processedPath)
	if err != nil {
		// Clean up original file if processing fails
		os.Remove(originalPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process audio"})
		return
	}

	// Store song metadata
	song := &Song{
		ID:        id,
		Name:      strings.TrimSuffix(header.Filename, ".mp3"),
		Original:  originalPath,
		Processed: processedPath,
		CreatedAt: time.Now(),
	}
	
	err = saveSong(song)
	if err != nil {
		// Clean up files if database save fails
		os.Remove(originalPath)
		os.Remove(processedPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save song metadata"})
		return
	}

	c.JSON(http.StatusOK, song)
}

func getSongs(c *gin.Context) {
	songList, err := getAllSongs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch songs"})
		return
	}
	c.JSON(http.StatusOK, songList)
}

func downloadSong(c *gin.Context) {
	id := c.Param("id")
	song, err := getSongByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_no_drums.mp3", song.Name))
	c.File(song.Processed)
}

func downloadOriginalSong(c *gin.Context) {
	id := c.Param("id")
	song, err := getSongByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_original.mp3", song.Name))
	c.File(song.Original)
}

func deleteSong(c *gin.Context) {
	id := c.Param("id")
	song, err := getSongByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	// Delete files
	os.Remove(song.Original)
	os.Remove(song.Processed)

	// Remove from database
	err = deleteSongFromDB(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete song from database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Song deleted successfully"})
}

func renameSong(c *gin.Context) {
	id := c.Param("id")
	song, err := getSongByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err = updateSongName(id, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update song name"})
		return
	}

	song.Name = req.Name
	c.JSON(http.StatusOK, song)
}

func removeDrums(inputPath, outputPath string) error {
	// Create temporary directory for Spleeter output
	tempDir := filepath.Join("temp", uuid.New().String())
	defer os.RemoveAll(tempDir)
	
	// Use Spleeter to separate stems (vocals, drums, bass, other)
	cmd := exec.Command("spleeter", "separate", "-p", "spleeter:4stems-16kHz", "-o", tempDir, inputPath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("spleeter separation failed: %w", err)
	}
	
	// Get the base filename without extension
	baseName := strings.TrimSuffix(filepath.Base(inputPath), ".mp3")
	
	// Combine vocals, bass, and other stems (excluding drums)
	vocalsPath := filepath.Join(tempDir, baseName, "vocals.wav")
	bassPath := filepath.Join(tempDir, baseName, "bass.wav")
	otherPath := filepath.Join(tempDir, baseName, "other.wav")
	
	// Use ffmpeg to mix the non-drum stems and convert to MP3
	cmd = exec.Command("ffmpeg", 
		"-i", vocalsPath,
		"-i", bassPath, 
		"-i", otherPath,
		"-filter_complex", "[0:a][1:a][2:a]amix=inputs=3:duration=longest:normalize=0",
		"-c:a", "mp3",
		"-b:a", "192k",
		"-y", outputPath)
	
	return cmd.Run()
}

func cleanupTempFiles() {
	// Clean up any leftover temporary directories
	tempDir := "temp"
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		log.Printf("Failed to read temp directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fullPath := filepath.Join(tempDir, entry.Name())
			err := os.RemoveAll(fullPath)
			if err != nil {
				log.Printf("Failed to remove temp directory %s: %v", fullPath, err)
			}
		}
	}
	log.Println("Cleaned up temporary files on startup")
}

func getVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": Version,
		"name":    "Drummer",
	})
}