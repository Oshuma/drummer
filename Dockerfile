# Build stage for React frontend
FROM node:18-alpine as frontend-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Build stage for Go backend
FROM golang:1.21 as backend-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

# Build with CGO enabled for SQLite support
ENV CGO_ENABLED=1
RUN go build -o drummer

# Final stage
FROM python:3.8-slim

# Install system dependencies for Spleeter and FFmpeg
RUN apt-get update && apt-get install -y \
    ffmpeg \
    libsndfile1 \
    libsndfile1-dev \
    build-essential \
    pkg-config \
    libasound2-dev \
    portaudio19-dev \
    libportaudio2 \
    libportaudiocpp0 \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Upgrade pip and install Python dependencies with compatible versions
RUN pip install --upgrade pip==20.3.4 setuptools==57.5.0 wheel==0.37.1

# Install compatible versions step by step - use older numpy for TF 2.5
RUN pip install numpy==1.18.5
RUN pip install tensorflow==2.5.0
RUN pip install librosa==0.8.1

# Install Spleeter - models will be downloaded at runtime
RUN pip install spleeter==2.3.2

WORKDIR /app

# Copy the Go binary
COPY --from=backend-builder /app/drummer .

# Copy the React build
COPY --from=frontend-builder /app/web/build ./web/build

# Create directories for uploads and database
RUN mkdir -p uploads processed temp data

# Expose port
EXPOSE 8080

# Run the application
CMD ["./drummer"]