package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/zaguerinho/claude-switch/agent-hub/internal/model"
)

const defaultBaseDir = "/.agent-hub"

// baseDir returns the agent-hub data directory.
func baseDir() string {
	home, _ := os.UserHomeDir()
	return home + defaultBaseDir
}

// resolveAPIPort returns the API port from flag, env, or default.
func resolveAPIPort() int {
	if apiPort != 0 {
		return apiPort
	}
	if v := os.Getenv("AGENT_HUB_PORT"); v != "" {
		var p int
		fmt.Sscanf(v, "%d", &p)
		if p > 0 {
			return p
		}
	}
	return 7777
}

// apiURL constructs a full API URL.
func apiURL(path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", resolveAPIPort(), path)
}

// apiCall makes an HTTP request to the API server and returns the response.
func apiCall(method, path string, body any) (*model.APIResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, apiURL(path), bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Auto-start server on connection refused
		if autoStartServer() {
			// Rebuild the request (body may have been consumed)
			if body != nil {
				data, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(data)
			}
			req, _ = http.NewRequest(method, apiURL(path), bodyReader)
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err = client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("server not reachable after auto-start: %w", err)
			}
		} else {
			return nil, fmt.Errorf("server not reachable (start with: agent-hub serve)")
		}
	}
	defer resp.Body.Close()

	var apiResp model.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &apiResp, nil
}

// printJSON outputs the raw APIResponse as formatted JSON.
func printJSON(resp *model.APIResponse) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(resp)
}

// printError prints an error to stderr and returns it.
func printError(msg string) error {
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(model.APIResponse{OK: false, Error: msg})
		return fmt.Errorf("%s", msg)
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	return fmt.Errorf("%s", msg)
}

// handleResponse processes an API response: prints JSON in json mode,
// or calls the formatter function for human output.
func handleResponse(resp *model.APIResponse, err error, formatter func(data any)) error {
	if err != nil {
		return printError(err.Error())
	}
	if !resp.OK {
		return printError(resp.Error)
	}
	if jsonOutput {
		printJSON(resp)
		return nil
	}
	if formatter != nil {
		formatter(resp.Data)
	}
	return nil
}

// autoStartServer attempts to start the server in the background.
// Returns true if the server was started successfully.
func autoStartServer() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}

	proc := exec.Command(exe, "serve", "--foreground",
		"--api-port", fmt.Sprintf("%d", resolveAPIPort()),
		"--ui-port", fmt.Sprintf("%d", uiPort),
	)
	proc.Stdout = nil
	proc.Stderr = nil
	proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := proc.Start(); err != nil {
		return false
	}

	// Wait for server to be ready
	client := &http.Client{Timeout: 500 * time.Millisecond}
	healthURL := apiURL("/api/v1/health")
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err := client.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "agent-hub: server auto-started (PID %d)\n", proc.Process.Pid)
			return true
		}
	}
	return false
}

// resolveAlias determines the agent alias from (in order):
// 1. Explicit flag value (if non-empty)
// 2. .agent-hub-alias file in current directory
// 3. ~/.agent-hub/identity.json (global identity)
// 4. AGENT_HUB_ALIAS env var
// Returns empty string if none found.
func resolveAlias(explicit string) string {
	if explicit != "" {
		return explicit
	}
	// Check .agent-hub-alias in cwd
	if data, err := os.ReadFile(".agent-hub-alias"); err == nil {
		if alias := strings.TrimSpace(string(data)); alias != "" {
			return alias
		}
	}
	// Check global identity
	home, _ := os.UserHomeDir()
	identPath := filepath.Join(home, defaultBaseDir, "identity.json")
	if data, err := os.ReadFile(identPath); err == nil {
		var id Identity
		if json.Unmarshal(data, &id) == nil && id.Alias != "" {
			return id.Alias
		}
	}
	// Check env var
	return os.Getenv("AGENT_HUB_ALIAS")
}
