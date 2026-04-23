package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

// CreateRoom initializes a new room directory with empty metadata.
func (fs *FileStore) CreateRoom(name, description string) (*model.RoomMeta, error) {
	mu := fs.mu.get(name)
	mu.Lock()
	defer mu.Unlock()

	dir := fs.roomDir(name)
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("room %q already exists", name)
	}

	if err := os.MkdirAll(filepath.Join(dir, "messages"), 0o755); err != nil {
		return nil, fmt.Errorf("create room dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		return nil, fmt.Errorf("create docs dir: %w", err)
	}

	meta := &model.RoomMeta{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UTC(),
	}
	if err := atomicWriteJSON(filepath.Join(dir, "meta.json"), meta); err != nil {
		return nil, err
	}

	// Initialize empty agents, cursors, and status
	if err := atomicWriteJSON(filepath.Join(dir, "agents.json"), &model.AgentList{}); err != nil {
		return nil, err
	}
	if err := atomicWriteJSON(filepath.Join(dir, "cursors.json"), &model.CursorMap{Cursors: map[string]int{}}); err != nil {
		return nil, err
	}
	if err := atomicWriteJSON(filepath.Join(dir, "status.json"), &model.StatusBoard{}); err != nil {
		return nil, err
	}

	return meta, nil
}

// GetRoom reads room metadata.
func (fs *FileStore) GetRoom(name string) (*model.RoomMeta, error) {
	mu := fs.mu.get(name)
	mu.Lock()
	defer mu.Unlock()

	return fs.getRoomUnlocked(name)
}

func (fs *FileStore) getRoomUnlocked(name string) (*model.RoomMeta, error) {
	var meta model.RoomMeta
	path := filepath.Join(fs.roomDir(name), "meta.json")
	if err := readJSON(path, &meta); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("room %q not found", name)
		}
		return nil, err
	}
	return &meta, nil
}

// ListRooms returns all non-archived rooms.
func (fs *FileStore) ListRooms() ([]model.RoomInfo, error) {
	dir := filepath.Join(fs.baseDir, "rooms")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var rooms []model.RoomInfo
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archived" {
			continue
		}
		info, err := fs.GetRoomInfo(e.Name())
		if err != nil {
			continue
		}
		if !info.Archived {
			rooms = append(rooms, *info)
		}
	}
	return rooms, nil
}

// ListAllRooms returns all rooms including archived.
func (fs *FileStore) ListAllRooms() ([]model.RoomInfo, error) {
	rooms, err := fs.ListRooms()
	if err != nil {
		return nil, err
	}

	archDir := filepath.Join(fs.baseDir, "rooms", "archived")
	entries, err := os.ReadDir(archDir)
	if err != nil {
		if os.IsNotExist(err) {
			return rooms, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := fs.getArchivedRoomInfo(e.Name())
		if err != nil {
			continue
		}
		rooms = append(rooms, *info)
	}
	return rooms, nil
}

// GetRoomInfo returns room metadata enriched with counts.
func (fs *FileStore) GetRoomInfo(name string) (*model.RoomInfo, error) {
	mu := fs.mu.get(name)
	mu.Lock()
	defer mu.Unlock()

	meta, err := fs.getRoomUnlocked(name)
	if err != nil {
		return nil, err
	}

	agents, _ := fs.getAgentsUnlocked(name)
	msgCount, _ := fs.messageCountUnlocked(name)
	lastAct := fs.lastActivityUnlocked(name)

	return &model.RoomInfo{
		RoomMeta:     *meta,
		AgentCount:   len(agents),
		MessageCount: msgCount,
		LastActivity: lastAct,
	}, nil
}

func (fs *FileStore) getArchivedRoomInfo(name string) (*model.RoomInfo, error) {
	var meta model.RoomMeta
	path := filepath.Join(fs.archivedRoomDir(name), "meta.json")
	if err := readJSON(path, &meta); err != nil {
		return nil, err
	}
	return &model.RoomInfo{RoomMeta: meta}, nil
}

// ArchiveRoom moves a room to the archived directory.
func (fs *FileStore) ArchiveRoom(name string) error {
	mu := fs.mu.get(name)
	mu.Lock()
	defer mu.Unlock()

	src := fs.roomDir(name)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("room %q not found", name)
	}

	// Update meta to mark as archived
	meta, err := fs.getRoomUnlocked(name)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	meta.Archived = true
	meta.ArchivedAt = &now
	if err := atomicWriteJSON(filepath.Join(src, "meta.json"), meta); err != nil {
		return err
	}

	dst := fs.archivedRoomDir(name)
	return os.Rename(src, dst)
}

// RoomExists checks whether a non-archived room exists.
func (fs *FileStore) RoomExists(name string) bool {
	_, err := os.Stat(filepath.Join(fs.roomDir(name), "meta.json"))
	return err == nil
}
