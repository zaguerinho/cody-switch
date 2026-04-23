package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

// PostMessage writes a new message with the next sequential ID.
func (fs *FileStore) PostMessage(room string, from string, msgType model.MessageType, subject, body string) (*model.Message, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	if _, err := fs.getRoomUnlocked(room); err != nil {
		return nil, err
	}

	dir := fs.messagesDir(room)
	nextID, err := fs.nextMessageIDUnlocked(dir)
	if err != nil {
		return nil, err
	}

	msg := &model.Message{
		ID:        nextID,
		From:      from,
		Type:      msgType,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now().UTC(),
	}

	filename := fmt.Sprintf("%04d.json", nextID)
	if err := atomicWriteJSON(filepath.Join(dir, filename), msg); err != nil {
		return nil, err
	}

	return msg, nil
}

// ReadMessages returns messages for a room with optional filtering.
func (fs *FileStore) ReadMessages(room string, last int) ([]model.Message, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	return fs.readMessagesUnlocked(room, last)
}

func (fs *FileStore) readMessagesUnlocked(room string, last int) ([]model.Message, error) {
	dir := fs.messagesDir(room)
	ids, err := fs.listMessageIDsUnlocked(dir)
	if err != nil {
		return nil, err
	}

	if last > 0 && len(ids) > last {
		ids = ids[len(ids)-last:]
	}

	messages := make([]model.Message, 0, len(ids))
	for _, id := range ids {
		var msg model.Message
		filename := fmt.Sprintf("%04d.json", id)
		if err := readJSON(filepath.Join(dir, filename), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// ReadUnread returns messages after the agent's cursor position.
func (fs *FileStore) ReadUnread(room, alias string) ([]model.Message, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	cursor, err := fs.getCursorUnlocked(room, alias)
	if err != nil {
		return nil, err
	}

	dir := fs.messagesDir(room)
	ids, err := fs.listMessageIDsUnlocked(dir)
	if err != nil {
		return nil, err
	}

	var unreadIDs []int
	for _, id := range ids {
		if id > cursor {
			unreadIDs = append(unreadIDs, id)
		}
	}

	messages := make([]model.Message, 0, len(unreadIDs))
	for _, id := range unreadIDs {
		var msg model.Message
		filename := fmt.Sprintf("%04d.json", id)
		if err := readJSON(filepath.Join(dir, filename), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// UnreadCount returns the number of unread messages for an agent.
func (fs *FileStore) UnreadCount(room, alias string) (int, *model.Message, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	cursor, err := fs.getCursorUnlocked(room, alias)
	if err != nil {
		return 0, nil, err
	}

	dir := fs.messagesDir(room)
	ids, err := fs.listMessageIDsUnlocked(dir)
	if err != nil {
		return 0, nil, err
	}

	count := 0
	for _, id := range ids {
		if id > cursor {
			count++
		}
	}

	var latest *model.Message
	if count > 0 && len(ids) > 0 {
		var msg model.Message
		filename := fmt.Sprintf("%04d.json", ids[len(ids)-1])
		if err := readJSON(filepath.Join(dir, filename), &msg); err == nil {
			latest = &msg
		}
	}

	return count, latest, nil
}

// MessageCount returns total message count for a room.
func (fs *FileStore) MessageCount(room string) (int, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	return fs.messageCountUnlocked(room)
}

func (fs *FileStore) messageCountUnlocked(room string) (int, error) {
	dir := fs.messagesDir(room)
	ids, err := fs.listMessageIDsUnlocked(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return len(ids), nil
}

// nextMessageIDUnlocked determines the next sequential ID.
func (fs *FileStore) nextMessageIDUnlocked(dir string) (int, error) {
	ids, err := fs.listMessageIDsUnlocked(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, err
	}
	if len(ids) == 0 {
		return 1, nil
	}
	return ids[len(ids)-1] + 1, nil
}

// listMessageIDsUnlocked returns sorted message IDs from the directory.
func (fs *FileStore) listMessageIDsUnlocked(dir string) ([]int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var ids []int
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".tmp") {
			continue
		}
		numStr := strings.TrimSuffix(name, ".json")
		id, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

// lastActivityUnlocked returns the timestamp of the most recent message.
func (fs *FileStore) lastActivityUnlocked(room string) *time.Time {
	dir := fs.messagesDir(room)
	ids, err := fs.listMessageIDsUnlocked(dir)
	if err != nil || len(ids) == 0 {
		return nil
	}

	var msg model.Message
	filename := fmt.Sprintf("%04d.json", ids[len(ids)-1])
	if err := readJSON(filepath.Join(dir, filename), &msg); err != nil {
		return nil
	}
	return &msg.Timestamp
}
