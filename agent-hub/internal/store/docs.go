package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DocInfo describes a document in a room.
type DocInfo struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// ScaffoldDocs writes governance templates into a room's docs/ directory.
// Existing files are never overwritten.
func (fs *FileStore) ScaffoldDocs(room string, templates map[string][]byte) error {
	dir := filepath.Join(fs.roomDir(room), "docs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, content := range templates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			continue // don't overwrite
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// ListDocs returns metadata for all docs in a room.
func (fs *FileStore) ListDocs(room string) ([]DocInfo, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	dir := filepath.Join(fs.roomDir(room), "docs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var docs []DocInfo
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		docs = append(docs, DocInfo{
			Name:     e.Name(),
			Size:     info.Size(),
			Modified: info.ModTime().UTC(),
		})
	}
	return docs, nil
}

// ReadDoc returns the content of a document.
func (fs *FileStore) ReadDoc(room, name string) (string, error) {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	path := filepath.Join(fs.roomDir(room), "docs", name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("doc %q not found in room %q", name, room)
		}
		return "", err
	}
	return string(data), nil
}

// WriteDoc writes or updates a document in a room.
func (fs *FileStore) WriteDoc(room, name, content string) error {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	dir := filepath.Join(fs.roomDir(room), "docs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return atomicWriteJSON_raw(filepath.Join(dir, name), []byte(content))
}

// AppendToStatusDoc appends an activity log entry to STATUS.md.
func (fs *FileStore) AppendToStatusDoc(room, key, value, updatedBy string) error {
	mu := fs.mu.get(room)
	mu.Lock()
	defer mu.Unlock()

	path := filepath.Join(fs.roomDir(room), "docs", "STATUS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no STATUS.md, skip silently
		}
		return err
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04 UTC")
	entry := fmt.Sprintf("- **%s** = %s _(by %s, %s)_\n", key, value, updatedBy, timestamp)

	content := string(data) + entry
	return os.WriteFile(path, []byte(content), 0o644)
}

// atomicWriteJSON_raw writes raw bytes using the temp-then-rename pattern.
func atomicWriteJSON_raw(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
