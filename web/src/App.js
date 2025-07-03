import React, { useState, useEffect } from 'react';
import './App.css';

function App() {
  const [songs, setSongs] = useState([]);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadFileName, setUploadFileName] = useState('');
  const [message, setMessage] = useState('');
  const [messageType, setMessageType] = useState('');
  const [editingId, setEditingId] = useState(null);
  const [editingName, setEditingName] = useState('');
  const [version, setVersion] = useState('');
  const [youtubeUrl, setYoutubeUrl] = useState('');

  useEffect(() => {
    fetchSongs();
    fetchVersion();
  }, []);

  const fetchSongs = async () => {
    try {
      const response = await fetch('/api/songs');
      const data = await response.json();
      setSongs(data || []);
    } catch (error) {
      showMessage('Failed to fetch songs', 'error');
    }
  };

  const fetchVersion = async () => {
    try {
      const response = await fetch('/api/version');
      const data = await response.json();
      setVersion(data.version);
    } catch (error) {
      console.error('Failed to fetch version:', error);
    }
  };

  const showMessage = (text, type) => {
    setMessage(text);
    setMessageType(type);
    setTimeout(() => {
      setMessage('');
      setMessageType('');
    }, 5000);
  };

  const handleFileUpload = async (file) => {
    if (!file) return;

    if (!file.name.toLowerCase().endsWith('.mp3')) {
      showMessage('Please upload an MP3 file', 'error');
      return;
    }

    setUploading(true);
    setUploadProgress(0);
    setUploadFileName(file.name);

    // Simulate upload progress steps
    const progressSteps = [
      { progress: 10, message: 'Uploading file...' },
      { progress: 30, message: 'Processing with AI...' },
      { progress: 60, message: 'Separating audio stems...' },
      { progress: 85, message: 'Removing drums...' },
      { progress: 95, message: 'Finalizing...' }
    ];

    const formData = new FormData();
    formData.append('file', file);

    try {
      // Start progress simulation
      let currentStep = 0;
      const progressInterval = setInterval(() => {
        if (currentStep < progressSteps.length) {
          setUploadProgress(progressSteps[currentStep].progress);
          showMessage(progressSteps[currentStep].message, 'info');
          currentStep++;
        }
      }, 2000);

      const response = await fetch('/api/upload', {
        method: 'POST',
        body: formData,
      });

      clearInterval(progressInterval);
      setUploadProgress(100);

      if (response.ok) {
        const newSong = await response.json();
        setSongs([...songs, newSong]);
        showMessage('Song uploaded and processed successfully!', 'success');
      } else {
        const error = await response.json();
        showMessage(error.error || 'Upload failed', 'error');
      }
    } catch (error) {
      showMessage('Upload failed', 'error');
    } finally {
      setUploading(false);
      setUploadProgress(0);
      setUploadFileName('');
    }
  };

  const handleYoutubeUpload = async () => {
    if (!youtubeUrl.trim()) {
      showMessage('Please enter a YouTube URL', 'error');
      return;
    }

    // Basic YouTube URL validation
    const youtubeRegex = /^(https?:\/\/)?(www\.)?(youtube\.com\/watch\?v=|youtu\.be\/)[\w-]+/;
    if (!youtubeRegex.test(youtubeUrl)) {
      showMessage('Please enter a valid YouTube URL', 'error');
      return;
    }

    setUploading(true);
    setUploadProgress(0);
    setUploadFileName('YouTube video');

    // YouTube processing steps
    const progressSteps = [
      { progress: 15, message: 'Downloading from YouTube...' },
      { progress: 35, message: 'Converting to audio...' },
      { progress: 55, message: 'Processing with AI...' },
      { progress: 75, message: 'Separating audio stems...' },
      { progress: 90, message: 'Removing drums...' },
      { progress: 95, message: 'Finalizing...' }
    ];

    try {
      // Start progress simulation
      let currentStep = 0;
      const progressInterval = setInterval(() => {
        if (currentStep < progressSteps.length) {
          setUploadProgress(progressSteps[currentStep].progress);
          showMessage(progressSteps[currentStep].message, 'info');
          currentStep++;
        }
      }, 3000);

      const response = await fetch('/api/youtube', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url: youtubeUrl }),
      });

      clearInterval(progressInterval);
      setUploadProgress(100);

      if (response.ok) {
        const newSong = await response.json();
        setSongs([...songs, newSong]);
        showMessage('YouTube video processed successfully!', 'success');
        setYoutubeUrl('');
      } else {
        const error = await response.json();
        showMessage(error.error || 'YouTube processing failed', 'error');
      }
    } catch (error) {
      showMessage('YouTube processing failed', 'error');
    } finally {
      setUploading(false);
      setUploadProgress(0);
      setUploadFileName('');
    }
  };

  const handleFileChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      handleFileUpload(file);
    }
  };

  const handleDrop = (e) => {
    e.preventDefault();
    e.stopPropagation();
    const file = e.dataTransfer.files[0];
    if (file) {
      handleFileUpload(file);
    }
  };

  const handleDragOver = (e) => {
    e.preventDefault();
    e.stopPropagation();
  };

  const handleDownload = (id, name) => {
    const link = document.createElement('a');
    link.href = `/api/download/${id}`;
    link.download = `${name}_no_drums.mp3`;
    link.click();
  };

  const handleDownloadOriginal = (id, name) => {
    const link = document.createElement('a');
    link.href = `/api/download/${id}/original`;
    link.download = `${name}_original.mp3`;
    link.click();
  };

  const handleDelete = async (id) => {
    if (window.confirm('Are you sure you want to delete this song?')) {
      try {
        const response = await fetch(`/api/songs/${id}`, {
          method: 'DELETE',
        });

        if (response.ok) {
          setSongs(songs.filter(song => song.id !== id));
          showMessage('Song deleted successfully', 'success');
        } else {
          showMessage('Failed to delete song', 'error');
        }
      } catch (error) {
        showMessage('Failed to delete song', 'error');
      }
    }
  };

  const handleRename = async (id) => {
    if (!editingName.trim()) {
      showMessage('Please enter a valid name', 'error');
      return;
    }

    try {
      const response = await fetch(`/api/songs/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name: editingName }),
      });

      if (response.ok) {
        const updatedSong = await response.json();
        setSongs(songs.map(song => song.id === id ? updatedSong : song));
        setEditingId(null);
        setEditingName('');
        showMessage('Song renamed successfully', 'success');
      } else {
        showMessage('Failed to rename song', 'error');
      }
    } catch (error) {
      showMessage('Failed to rename song', 'error');
    }
  };

  const startEditing = (id, currentName) => {
    setEditingId(id);
    setEditingName(currentName);
  };

  const cancelEditing = () => {
    setEditingId(null);
    setEditingName('');
  };

  return (
    <div className="container">
      <div className="header">
        <h1>Drummer</h1>
        <p>Upload your favorite songs and practice without drums</p>
      </div>

      {/* Fixed toast messages */}
      {message && (
        <div className="toast-container">
          <div className={`toast ${messageType === 'error' ? 'error' : messageType === 'info' ? 'info' : 'success'}`}>
            {message}
          </div>
        </div>
      )}

      {uploading && (
        <div className="progress-section">
          <div className="progress-info">
            <span className="progress-filename">{uploadFileName}</span>
            <span className="progress-percentage">{uploadProgress}%</span>
          </div>
          <div className="progress-bar">
            <div 
              className="progress-fill" 
              style={{ width: `${uploadProgress}%` }}
            ></div>
          </div>
        </div>
      )}

      <div className="main-content">
        <div className="upload-section">
        <h2>Add Song</h2>
        
        {/* File Upload */}
        <div className="upload-method">
          <h3>Upload MP3 File</h3>
          <div
            className="upload-area"
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            onClick={() => document.getElementById('fileInput').click()}
          >
            <p>Drag and drop an MP3 file here or click to select</p>
            <input
              id="fileInput"
              type="file"
              accept=".mp3"
              onChange={handleFileChange}
              style={{ display: 'none' }}
            />
            <button className="upload-button" disabled={uploading}>
              {uploading ? 'Processing...' : 'Select File'}
            </button>
          </div>
        </div>

        {/* YouTube URL */}
        <div className="upload-method">
          <h3>YouTube URL</h3>
          <div className="youtube-input">
            <input
              type="url"
              placeholder="https://www.youtube.com/watch?v=..."
              value={youtubeUrl}
              onChange={(e) => setYoutubeUrl(e.target.value)}
              className="youtube-url-input"
              disabled={uploading}
            />
            <button 
              className="upload-button youtube-button" 
              onClick={handleYoutubeUpload}
              disabled={uploading || !youtubeUrl.trim()}
            >
              {uploading ? 'Processing...' : 'Process YouTube'}
            </button>
          </div>
        </div>
        </div>

        <div className="songs-section">
        <h2>Your Songs</h2>
        {songs.length === 0 ? (
          <p>No songs uploaded yet. Upload your first MP3 file above!</p>
        ) : (
          <table className="songs-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Upload Date</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {songs.map((song) => (
                <tr key={song.id}>
                  <td>
                    {editingId === song.id ? (
                      <input
                        type="text"
                        value={editingName}
                        onChange={(e) => setEditingName(e.target.value)}
                        className="rename-input"
                        onKeyPress={(e) => {
                          if (e.key === 'Enter') {
                            handleRename(song.id);
                          }
                        }}
                      />
                    ) : (
                      song.name
                    )}
                  </td>
                  <td>{new Date(song.created_at).toLocaleDateString()}</td>
                  <td>
                    {editingId === song.id ? (
                      <>
                        <button
                          className="action-button save"
                          onClick={() => handleRename(song.id)}
                          title="Save"
                        >
                          ✓
                        </button>
                        <button
                          className="action-button cancel"
                          onClick={cancelEditing}
                          title="Cancel"
                        >
                          ✕
                        </button>
                      </>
                    ) : (
                      <>
                        <button
                          className="action-button download"
                          onClick={() => handleDownload(song.id, song.name)}
                          title="Download (No Drums)"
                        >
                          ⬇
                        </button>
                        <button
                          className="action-button download-original"
                          onClick={() => handleDownloadOriginal(song.id, song.name)}
                          title="Download Original"
                        >
                          📁
                        </button>
                        <button
                          className="action-button rename"
                          onClick={() => startEditing(song.id, song.name)}
                          title="Rename"
                        >
                          ✏
                        </button>
                        <button
                          className="action-button delete"
                          onClick={() => handleDelete(song.id)}
                          title="Delete"
                        >
                          🗑
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        </div>
      </div>
      
      {/* Version display */}
      {version && (
        <div className="version-display">
          v{version}
        </div>
      )}
    </div>
  );
}

export default App;