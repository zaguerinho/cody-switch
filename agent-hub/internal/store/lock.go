package store

import (
	"encoding/json"
	"os"
	"sync"
)

// sessionMu provides per-room mutex guarding all file operations.
type sessionMu struct {
	locks sync.Map // map[string]*sync.Mutex
}

func (sm *sessionMu) get(session string) *sync.Mutex {
	v, _ := sm.locks.LoadOrStore(session, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// atomicWriteJSON writes v as indented JSON to path using write-tmp-then-rename.
func atomicWriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// readJSON reads and unmarshals a JSON file into v.
func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
