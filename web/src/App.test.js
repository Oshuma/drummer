import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { act } from 'react';
import '@testing-library/jest-dom';
import App from './App';

// No fake timers needed for these tests

beforeEach(() => {
  jest.clearAllMocks();
  // Mock both fetch calls that happen in the useEffect hook
  global.fetch = jest.fn((url) => {
    if (url === '/api/version') {
      return Promise.resolve({
        json: () => Promise.resolve({ version: 'test-version' }),
        ok: true,
      });
    }
    if (url === '/api/songs') {
      return Promise.resolve({
        json: () => Promise.resolve([]), // Default to no songs
        ok: true,
      });
    }
    return Promise.reject(new Error(`Unhandled fetch for: ${url}`));
  });
});

test('renders Drummer title and version', async () => {
  render(<App />);
  // Use findBy to wait for async operations triggered by useEffect
  expect(await screen.findByText(/Drummer/i)).toBeInTheDocument();
  expect(await screen.findByText(/vtest-version/i)).toBeInTheDocument();
});

test('fetches and displays songs on initial render', async () => {
  const mockSongs = [
    { id: '1', name: 'Test Song 1', created_at: new Date().toISOString() },
    { id: '2', name: 'Test Song 2', created_at: new Date().toISOString() },
  ];

  // Override the default fetch mock for this specific test
  global.fetch.mockImplementation((url) => {
    if (url === '/api/songs') {
      return Promise.resolve({
        json: () => Promise.resolve(mockSongs),
        ok: true,
      });
    }
    if (url === '/api/version') {
      return Promise.resolve({
        json: () => Promise.resolve({ version: 'test-version' }),
        ok: true,
      });
    }
    return Promise.reject(new Error(`Unhandled fetch for: ${url}`));
  });

  render(<App />);
  expect(await screen.findByText('Test Song 1')).toBeInTheDocument();
  expect(screen.getByText('Test Song 2')).toBeInTheDocument();
});

test('shows error message when fetching songs fails', async () => {
  // Mock a failed songs fetch
  global.fetch.mockImplementation((url) => {
    if (url === '/api/songs') {
      return Promise.reject(new Error('Network error'));
    }
    if (url === '/api/version') {
      return Promise.resolve({
        json: () => Promise.resolve({ version: 'test-version' }),
        ok: true,
      });
    }
    return Promise.reject(new Error(`Unhandled fetch for: ${url}`));
  });

  render(<App />);
  expect(await screen.findByText(/Failed to fetch songs/i)).toBeInTheDocument();
});

test('shows message when no songs are available', async () => {
  render(<App />);
  expect(await screen.findByText(/No songs uploaded yet/i)).toBeInTheDocument();
});

test('shows error on uploading non-mp3 file', async () => {
  render(<App />);
  // Wait for initial render to complete
  expect(await screen.findByText(/No songs uploaded yet/i)).toBeInTheDocument();

  const file = new File(['(⌐□_□)'], 'chucknorris.png', { type: 'image/png' });
  const fileInput = screen.getByTestId('fileInput');

  // No need to click the button, just fire the change event on the input
  fireEvent.change(fileInput, { target: { files: [file] } });

  expect(await screen.findByText(/Please upload an MP3 file/i)).toBeInTheDocument();
});

test('shows error on submitting empty youtube url', async () => {
  render(<App />);
  // Wait for initial render to complete
  expect(await screen.findByText(/No songs uploaded yet/i)).toBeInTheDocument();

  const youtubeButton = screen.getByRole('button', { name: /Process YouTube/i });
  // The button is disabled by default when the input is empty
  expect(youtubeButton).toBeDisabled();

  // It should remain disabled even after clicking
  fireEvent.click(youtubeButton);
  expect(youtubeButton).toBeDisabled();
});

test('shows error on submitting invalid youtube url', async () => {
  render(<App />);
  // Wait for initial render to complete
  expect(await screen.findByText(/No songs uploaded yet/i)).toBeInTheDocument();

  const youtubeInput = screen.getByPlaceholderText(/https:\/\/www\.youtube\.com\/watch\?v=/i);
  const youtubeButton = screen.getByRole('button', { name: /Process YouTube/i });

  // Use act for user events that cause state updates
  await act(async () => {
    fireEvent.change(youtubeInput, { target: { value: 'invalid-url' } });
    fireEvent.click(youtubeButton);
  });

  expect(await screen.findByText(/Please enter a valid YouTube URL/i)).toBeInTheDocument();
});