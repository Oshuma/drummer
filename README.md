# Drummer

> Generate an image of a madman playing drums. It should be chaotic and nihilistic.

![Drummer](drummer.png)

Drummer is an app used to practice your favorite songs. Upload an MP3 file or provide a YouTube URL and it will strip drums from that song using AI-powered source separation and provide an updated MP3 without the drums.

It provides a simple, but elegant, web UI that allows for uploading files, downloading from YouTube, and managing songs, as well as a table list of all previously processed and stripped songs, with the ability to delete and rename songs.

## Audio Processing

Drummer uses **Spleeter**, Deezer's AI-powered source separation library, to accurately separate drums from your music. Spleeter uses deep learning to isolate different instruments and vocals, providing much higher quality drum removal compared to traditional center-channel extraction methods.

The application uses Spleeter's **5-stem model** for highest fidelity separation into:
- Vocals
- Drums  
- Bass
- Piano
- Other instruments

The final output combines vocals, bass, piano, and other instruments while excluding the drums, giving you a clean, high-fidelity backing track for practice. Audio is processed using the highest quality settings to preserve acoustic accuracy.

## Architecture

### Backend

Written in Go and deployable in a Docker container. Uses Spleeter for AI-powered drum separation and FFmpeg for audio processing. Supports both file uploads and YouTube URL downloads using yt-dlp. Song metadata is persisted using SQLite database for tracking across container restarts.

### Frontend

Written in React and included with the deployed Docker container.

### Data Persistence

- **Database**: SQLite database stores song metadata (name, file paths, upload date)
- **Files**: Original and processed audio files are stored in mounted volumes
- **Volumes**: All data persists across container restarts via Docker volumes

## Getting Started

### Prerequisites

- Docker and Docker Compose

### Running the Application

1. Clone this repository
2. Run the application:
   ```bash
   docker-compose up --build
   ```
3. Access the application at `http://localhost:8080`

The first run may take longer as Spleeter downloads its pre-trained models when processing the first song.

### Usage

Once the application is running, you can:

1. **Upload MP3 files**: Click the upload button to select and upload MP3 files from your computer
2. **Download from YouTube**: Enter a YouTube URL to download and process the audio
3. **Manage songs**: View, rename, delete, and download your processed songs from the songs table

Both uploaded files and YouTube downloads are processed through the same high-quality drum removal pipeline.

### Data Storage

The application creates the following directories on your host machine:
- `./uploads/` - Original MP3 files
- `./processed/` - Processed MP3 files without drums  
- `./data/` - SQLite database file
- `./temp/` - Temporary files during processing

All your songs and metadata will persist across container restarts and rebuilds.
