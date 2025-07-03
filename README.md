# Drummer

Drummer is an app used to practice your favorite songs. Upload an MP3 and it will
strip drums from that song using AI-powered source separation and provide an updated MP3 without the drums.

It provides a simple, but elegant, web UI that allows for uploading and downloading
songs, as well as a table list of all previously uploaded and stripped songs, with
the ability to delete and rename songs.

## Audio Processing

Drummer uses **Spleeter**, Deezer's AI-powered source separation library, to accurately separate drums from your music. Spleeter uses deep learning to isolate different instruments and vocals, providing much higher quality drum removal compared to traditional center-channel extraction methods.

The application uses Spleeter's 4-stem model to separate audio into:
- Vocals
- Drums  
- Bass
- Other instruments

The final output combines vocals, bass, and other instruments while excluding the drums, giving you a clean backing track for practice.

## Architecture

### Backend

Written in Go and deployable in a Docker container. Uses Spleeter for AI-powered drum separation and FFmpeg for audio processing. Song metadata is persisted using SQLite database for tracking across container restarts.

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

### Data Storage

The application creates the following directories on your host machine:
- `./uploads/` - Original MP3 files
- `./processed/` - Processed MP3 files without drums  
- `./data/` - SQLite database file
- `./temp/` - Temporary files during processing

All your songs and metadata will persist across container restarts and rebuilds.
