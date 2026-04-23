package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

// AskRequest is the JSON body sent by the browser chatbot.
type AskRequest struct {
	Question      string           `json:"question"`
	ChapterTitle  string           `json:"chapterTitle"`
	Narration     string           `json:"narration"`
	File          string           `json:"file,omitempty"`
	Lines         [2]int           `json:"lines,omitempty"`
	CodeSnippet   string           `json:"codeSnippet,omitempty"`
	TutorialTitle string           `json:"tutorialTitle,omitempty"`
	History       []HistoryMessage `json:"history,omitempty"`
}

// HistoryMessage represents a single turn in the conversation.
type HistoryMessage struct {
	Role string `json:"role"` // "user" or "assistant"
	Text string `json:"text"`
}

// handleAsk returns a handler that streams Claude CLI responses as SSE.
func handleAsk() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		if req.Question == "" {
			http.Error(w, `{"error":"question is required"}`, http.StatusBadRequest)
			return
		}

		prompt := buildPrompt(req)

		// Set SSE headers before starting the subprocess.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
			return
		}

		// Start claude -p subprocess. Use request context so the process
		// is killed if the browser disconnects.
		cmd := exec.CommandContext(r.Context(), "claude", "-p", "--output-format", "text", prompt)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			writeSSEError(w, flusher, "failed to create pipe: "+err.Error())
			return
		}
		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf

		if err := cmd.Start(); err != nil {
			writeSSEError(w, flusher, "failed to start claude: "+err.Error())
			return
		}

		// Stream stdout chunks as SSE events.
		scanner := bufio.NewScanner(stdout)
		scanner.Split(scanChunks)

		for scanner.Scan() {
			chunk := scanner.Text()
			if chunk == "" {
				continue
			}
			writeSSEText(w, flusher, chunk)
		}

		// Wait for the process to finish and check exit code.
		if err := cmd.Wait(); err != nil {
			errMsg := stderrBuf.String()
			if errMsg == "" {
				errMsg = err.Error()
			}
			writeSSEError(w, flusher, "claude error: "+errMsg)
			return
		}

		writeSSEDone(w, flusher)
	}
}

// buildPrompt constructs the full prompt string from the request.
// It includes a system-level preamble, conversation history, and the current question.
func buildPrompt(req AskRequest) string {
	var b strings.Builder

	// System context
	b.WriteString("You are a helpful coding tutor embedded in a narrated code tutorial. ")
	b.WriteString("The user has paused the tutorial to ask a question.\n\n")

	if req.TutorialTitle != "" {
		fmt.Fprintf(&b, "Tutorial: %q\n", req.TutorialTitle)
	}
	fmt.Fprintf(&b, "Current chapter: %q\n", req.ChapterTitle)

	if req.File != "" {
		if req.Lines[0] > 0 && req.Lines[1] > 0 {
			fmt.Fprintf(&b, "Current file: %s (lines %d-%d)\n", req.File, req.Lines[0], req.Lines[1])
		} else {
			fmt.Fprintf(&b, "Current file: %s\n", req.File)
		}
	}

	if req.Narration != "" {
		fmt.Fprintf(&b, "\nNarration at this point:\n%s\n", req.Narration)
	}

	if req.CodeSnippet != "" {
		fmt.Fprintf(&b, "\nCode being discussed:\n```\n%s\n```\n", req.CodeSnippet)
	}

	b.WriteString("\nKeep answers concise and focused on the code being shown. ")
	b.WriteString("Use code examples when helpful. If the question is unrelated to the tutorial, gently redirect.\n")

	// Conversation history
	if len(req.History) > 0 {
		b.WriteString("\n--- Conversation so far ---\n")
		for _, msg := range req.History {
			if msg.Text != "" {
				role := msg.Role
			if len(role) > 0 {
				role = strings.ToUpper(role[:1]) + role[1:]
			}
			fmt.Fprintf(&b, "%s: %s\n", role, msg.Text)
			}
		}
		b.WriteString("--- End of conversation ---\n")
	}

	// Current question
	fmt.Fprintf(&b, "\nUser's question: %s", req.Question)

	return b.String()
}

// scanChunks is a bufio.SplitFunc that yields chunks of up to 256 bytes,
// preferring to break at newlines for readability.
func scanChunks(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Try to break at a newline within the first 256 bytes.
	limit := 256
	if limit > len(data) {
		limit = len(data)
	}

	if idx := indexOf(data[:limit], '\n'); idx >= 0 {
		return idx + 1, data[:idx+1], nil
	}

	// No newline found — if we have enough data, yield the chunk.
	if len(data) >= 256 {
		return 256, data[:256], nil
	}

	// At EOF, yield whatever remains.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

func indexOf(data []byte, b byte) int {
	for i, c := range data {
		if c == b {
			return i
		}
	}
	return -1
}

// SSE helpers

func writeSSEText(w http.ResponseWriter, f http.Flusher, text string) {
	escaped, _ := json.Marshal(text)
	fmt.Fprintf(w, "data: {\"text\":%s}\n\n", escaped)
	f.Flush()
}

func writeSSEDone(w http.ResponseWriter, f http.Flusher) {
	fmt.Fprintf(w, "data: {\"done\":true}\n\n")
	f.Flush()
}

func writeSSEError(w http.ResponseWriter, f http.Flusher, msg string) {
	escaped, _ := json.Marshal(msg)
	fmt.Fprintf(w, "data: {\"error\":%s}\n\n", escaped)
	f.Flush()
}
