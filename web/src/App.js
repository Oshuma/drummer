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

  useEffect(() => {
    fetchSongs();
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

      {message && (
        <div className={messageType === 'error' ? 'error' : messageType === 'info' ? 'info' : 'success'}>
          {message}
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

      <div className="upload-section">
        <h2>Upload MP3 File</h2>
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
                          className="action-button"
                          onClick={() => handleRename(song.id)}
                        >
                          Save
                        </button>
                        <button
                          className="action-button"
                          onClick={cancelEditing}
                        >
                          Cancel
                        </button>
                      </>
                    ) : (
                      <>
                        <button
                          className="action-button"
                          onClick={() => handleDownload(song.id, song.name)}
                        >
                          Download
                        </button>
                        <button
                          className="action-button"
                          onClick={() => startEditing(song.id, song.name)}
                        >
                          Rename
                        </button>
                        <button
                          className="action-button delete"
                          onClick={() => handleDelete(song.id)}
                        >
                          Delete
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
  );
}

export default App;