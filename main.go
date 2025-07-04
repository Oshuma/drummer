package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"math/rand"
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

const Version = "0.3.0"

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
}

func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

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
		api.POST("/youtube", downloadYoutube)
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
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/songs.db"
	}
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Create data directory
	os.MkdirAll(filepath.Dir(dbPath), 0755)

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

	// Return an empty array if the list is nil
	if songList == nil {
		c.JSON(http.StatusOK, make([]*Song, 0))
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

	// Use Spleeter's highest fidelity 5-stem model for better separation
	cmd := exec.Command("spleeter", "separate",
		"-p", "spleeter:5stems-16kHz",
		"-o", tempDir,
		inputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Spleeter separation failed: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("spleeter separation failed: %w", err)
	}

	// Get the base filename without extension
	baseName := strings.TrimSuffix(filepath.Base(inputPath), ".mp3")

	// 5-stem model provides: vocals, drums, bass, piano, other
	vocalsPath := filepath.Join(tempDir, baseName, "vocals.wav")
	bassPath := filepath.Join(tempDir, baseName, "bass.wav")
	pianoPath := filepath.Join(tempDir, baseName, "piano.wav")
	otherPath := filepath.Join(tempDir, baseName, "other.wav")

	// Use high-quality FFmpeg settings for mixing and encoding
	cmd = exec.Command("ffmpeg",
		"-i", vocalsPath,
		"-i", bassPath,
		"-i", pianoPath,
		"-i", otherPath,
		"-filter_complex", "[0:a][1:a][2:a][3:a]amix=inputs=4:duration=longest:normalize=0:weights=1 1 1 1",
		"-c:a", "libmp3lame",
		"-q:a", "0", // Highest quality VBR
		"-ar", "44100", // Standard sample rate
		"-ac", "2", // Stereo
		"-y", outputPath)

	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg mixing failed: %v\nOutput: %s", err, string(output))
		return fmt.Errorf("audio mixing failed: %w", err)
	}

	return nil
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

func downloadYoutubeWithRetry(url string, tempDir string, tempAudioPath string, maxRetries int) error {
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("YouTube download attempt %d/%d for URL: %s", attempt, maxRetries, url)

		// Get random user agent for this attempt
		userAgent := getRandomUserAgent()

		// Build yt-dlp command with anti-detection measures
		cmd := exec.Command("yt-dlp",
			"--extract-audio",
			"--audio-format", "mp3",
			"--audio-quality", "192K",
			"--output", tempAudioPath,
			"--no-playlist",
			"--user-agent", userAgent,
			"--add-header", "Accept:text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
			"--add-header", "Accept-Language:en-US,en;q=0.5",
			"--add-header", "Accept-Encoding:gzip, deflate, br",
			"--add-header", "DNT:1",
			"--add-header", "Connection:keep-alive",
			"--add-header", "Upgrade-Insecure-Requests:1",
			"--sleep-interval", "1",
			"--max-sleep-interval", "5",
			"--verbose",
			url)

		// Capture both stdout and stderr
		output, err := cmd.CombinedOutput()

		if err == nil {
			log.Printf("YouTube download successful on attempt %d", attempt)
			return nil
		}

		lastError = fmt.Errorf("attempt %d failed: %v, output: %s", attempt, err, string(output))
		log.Printf("YouTube download attempt %d failed: %v", attempt, lastError)

		// Don't sleep after the last attempt
		if attempt < maxRetries {
			// Exponential backoff with jitter
			sleepTime := time.Duration(attempt*attempt) * time.Second
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			totalSleep := sleepTime + jitter

			log.Printf("Waiting %v before retry...", totalSleep)
			time.Sleep(totalSleep)
		}
	}

	return fmt.Errorf("all %d attempts failed, last error: %v", maxRetries, lastError)
}

func downloadYoutube(c *gin.Context) {
	var req struct {
		URL string `json:"url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "YouTube URL is required"})
		return
	}

	// Generate unique ID for this download
	id := uuid.New().String()
	tempDir := filepath.Join("temp", id)
	tempAudioPath := filepath.Join(tempDir, "%(title)s.%(ext)s")
	originalPath := filepath.Join("uploads", id+".mp3")
	processedPath := filepath.Join("processed", id+".mp3")

	// Create temp directory for this download
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}
	defer os.RemoveAll(tempDir)

	// Download with retry logic
	err = downloadYoutubeWithRetry(req.URL, tempDir, tempAudioPath, 3)
	if err != nil {
		log.Printf("YouTube download failed after all retries: %v", err)

		// Provide more specific error messages
		errorMsg := "Failed to download from YouTube"
		if strings.Contains(err.Error(), "network") || strings.Contains(err.Error(), "connection") {
			errorMsg = "Network error: Unable to connect to YouTube"
		} else if strings.Contains(err.Error(), "permission") || strings.Contains(err.Error(), "forbidden") {
			errorMsg = "Permission error: Video may be private or restricted"
		} else if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
			errorMsg = "Video not found: Please check the URL"
		} else if strings.Contains(err.Error(), "age") || strings.Contains(err.Error(), "login") {
			errorMsg = "Video is age-restricted or requires login"
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
		return
	}

	// Find the downloaded file in the temp directory
	files, err := os.ReadDir(tempDir)
	if err != nil || len(files) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Downloaded file not found"})
		return
	}

	// Get the first (and should be only) file
	downloadedFile := filepath.Join(tempDir, files[0].Name())

	// Copy the downloaded file to uploads directory (handle cross-device links)
	err = copyFile(downloadedFile, originalPath)
	if err != nil {
		log.Printf("Failed to copy file from %s to %s: %v", downloadedFile, originalPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process downloaded file"})
		return
	}

	// Remove the temporary file after successful copy
	os.Remove(downloadedFile)

	// Process the file to remove drums
	err = removeDrums(originalPath, processedPath)
	if err != nil {
		// Clean up original file if processing fails
		os.Remove(originalPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process audio"})
		return
	}

	// Get video title for the song name
	titleCmd := exec.Command("yt-dlp", "--get-title", "--no-playlist", "--user-agent", getRandomUserAgent(), req.URL)
	titleOutput, err := titleCmd.Output()
	songName := "YouTube Video"
	if err == nil {
		songName = strings.TrimSpace(string(titleOutput))
		// Clean up title for filesystem safety
		songName = strings.ReplaceAll(songName, "/", "-")
		songName = strings.ReplaceAll(songName, "\\", "-")
		if len(songName) > 100 {
			songName = songName[:100]
		}
	}

	// Store song metadata
	song := &Song{
		ID:        id,
		Name:      songName,
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

func getVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": Version,
		"name":    "Drummer",
	})
}
