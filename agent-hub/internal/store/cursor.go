package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

// Acknowledge updates the read cursor for an agent. If upTo is 0,
// acknowledges up to the latest message.
func (fs *FileStore) Acknowledge(room, alias string, upTo int) error {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	// Validate the target ID
	dir := fs.messagesDir(room)
	ids, err := fs.listMessageIDsUnlocked(dir)
	if err != nil {
		return err
	}

	maxID := 0
	if len(ids) > 0 {
		maxID = ids[len(ids)-1]
	}

	if upTo == 0 {
		upTo = maxID
	}
	if upTo > maxID {
		return fmt.Errorf("cannot ack up to %d: latest message is %d", upTo, maxID)
	}

	return fs.setCursorUnlocked(room, alias, upTo)
}

// GetCursor returns the last-read message ID for an agent.
func (fs *FileStore) GetCursor(room, alias string) (int, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	return fs.getCursorUnlocked(room, alias)
}

func (fs *FileStore) getCursorUnlocked(room, alias string) (int, error) {
	path := filepath.Join(fs.roomDir(room), "cursors.json")
	var cm model.CursorMap
	if err := readJSON(path, &cm); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return cm.Cursors[alias], nil
}

func (fs *FileStore) setCursorUnlocked(room, alias string, id int) error {
	path := filepath.Join(fs.roomDir(room), "cursors.json")
	var cm model.CursorMap
	if err := readJSON(path, &cm); err != nil {
		if os.IsNotExist(err) {
			cm.Cursors = map[string]int{}
		} else {
			return err
		}
	}
	if cm.Cursors == nil {
		cm.Cursors = map[string]int{}
	}
	cm.Cursors[alias] = id
	return atomicWriteJSON(path, &cm)
}

// GetAllCursors returns all read cursors for a room.
func (fs *FileStore) GetAllCursors(room string) (map[string]int, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	path := filepath.Join(fs.roomDir(room), "cursors.json")
	var cm model.CursorMap
	if err := readJSON(path, &cm); err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, err
	}
	if cm.Cursors == nil {
		return map[string]int{}, nil
	}
	return cm.Cursors, nil
}
