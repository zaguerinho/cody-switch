package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PIDFile manages the server's PID file.
type PIDFile struct {
	path string
}

// NewPIDFile creates a PID file manager at the given base directory.
func NewPIDFile(baseDir string) *PIDFile {
	return &PIDFile{path: filepath.Join(baseDir, "hub.pid")}
}

// Write writes the current process PID to the file.
func (p *PIDFile) Write(pid int) error {
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p.path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

// Read returns the PID from the file, or 0 if the file doesn't exist.
func (p *PIDFile) Read() (int, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// IsRunning checks whether the PID in the file corresponds to a running process.
func (p *PIDFile) IsRunning() (bool, int) {
	pid, err := p.Read()
	if err != nil || pid == 0 {
		return false, 0
	}

	// signal 0 checks if process exists without actually sending a signal
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, pid
	}
	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		return false, pid
	}
	return true, pid
}

// Remove deletes the PID file.
func (p *PIDFile) Remove() error {
	return os.Remove(p.path)
}

// RemoveIfStale removes the PID file if the recorded process is not running.
func (p *PIDFile) RemoveIfStale() {
	running, _ := p.IsRunning()
	if !running {
		p.Remove()
	}
}

// Path returns the PID file path.
func (p *PIDFile) Path() string {
	return p.path
}

// String returns a human-readable status.
func (p *PIDFile) String() string {
	running, pid := p.IsRunning()
	if running {
		return fmt.Sprintf("running (PID %d)", pid)
	}
	if pid > 0 {
		return fmt.Sprintf("stale (PID %d not running)", pid)
	}
	return "not running"
}
