package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileStore provides file-based persistence for agent-hub rooms.
// All operations on a room are serialized via per-room mutexes.
type FileStore struct {
	baseDir string
	mu      sessionMu
}

// New creates a FileStore rooted at baseDir and ensures the directory exists.
func New(baseDir string) (*FileStore, error) {
	sessDir := filepath.Join(baseDir, "rooms")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		return nil, fmt.Errorf("create rooms dir: %w", err)
	}
	archDir := filepath.Join(sessDir, "archived")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		return nil, fmt.Errorf("create archived dir: %w", err)
	}
	fs := &FileStore{baseDir: baseDir}
	return fs, nil
}

// Init performs startup recovery: removes orphaned .tmp files.
func (fs *FileStore) Init() error {
	return filepath.Walk(filepath.Join(fs.baseDir, "rooms"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tmp") {
			os.Remove(path)
		}
		return nil
	})
}

// roomDir returns the path for a room's data directory.
func (fs *FileStore) roomDir(name string) string {
	return filepath.Join(fs.baseDir, "rooms", name)
}

// archivedRoomDir returns the path for an archived room.
func (fs *FileStore) archivedRoomDir(name string) string {
	return filepath.Join(fs.baseDir, "rooms", "archived", name)
}

// messagesDir returns the messages subdirectory for a room.
func (fs *FileStore) messagesDir(name string) string {
	return filepath.Join(fs.roomDir(name), "messages")
}
