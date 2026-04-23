package store

import (
	"os"
	"path/filepath"
	"time"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

// GetStatus returns the status board for a room.
func (fs *FileStore) GetStatus(room string) (*model.StatusBoard, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	return fs.getStatusUnlocked(room)
}

func (fs *FileStore) getStatusUnlocked(room string) (*model.StatusBoard, error) {
	path := filepath.Join(fs.roomDir(room), "status.json")
	var board model.StatusBoard
	if err := readJSON(path, &board); err != nil {
		if os.IsNotExist(err) {
			return &model.StatusBoard{}, nil
		}
		return nil, err
	}
	return &board, nil
}

// UpdateStatus sets a key-value pair on the status board.
func (fs *FileStore) UpdateStatus(room, key, value, updatedBy string) error {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	board, err := fs.getStatusUnlocked(room)
	if err != nil {
		return err
	}

	entry := model.StatusEntry{
		Key:       key,
		Value:     value,
		UpdatedBy: updatedBy,
		UpdatedAt: time.Now().UTC(),
	}

	// Update existing or append
	found := false
	for i, e := range board.Entries {
		if e.Key == key {
			board.Entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		board.Entries = append(board.Entries, entry)
	}

	path := filepath.Join(fs.roomDir(room), "status.json")
	return atomicWriteJSON(path, board)
}
