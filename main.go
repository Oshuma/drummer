package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Song struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Original  string    `json:"original"`
	Processed string    `json:"processed"`
	CreatedAt time.Time `json:"created_at"`
}

var songs = make(map[string]*Song)

func main() {
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
		api.DELETE("/songs/:id", deleteSong)
		api.PUT("/songs/:id", renameSong)
	}

	r.Run(":8080")
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Process the file to remove drums
	err = removeDrums(originalPath, processedPath)
	if err != nil {
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
	songs[id] = song

	c.JSON(http.StatusOK, song)
}

func getSongs(c *gin.Context) {
	var songList []*Song
	for _, song := range songs {
		songList = append(songList, song)
	}
	c.JSON(http.StatusOK, songList)
}

func downloadSong(c *gin.Context) {
	id := c.Param("id")
	song, exists := songs[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_no_drums.mp3", song.Name))
	c.File(song.Processed)
}

func deleteSong(c *gin.Context) {
	id := c.Param("id")
	song, exists := songs[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	// Delete files
	os.Remove(song.Original)
	os.Remove(song.Processed)

	// Remove from memory
	delete(songs, id)

	c.JSON(http.StatusOK, gin.H{"message": "Song deleted successfully"})
}

func renameSong(c *gin.Context) {
	id := c.Param("id")
	song, exists := songs[id]
	if !exists {
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