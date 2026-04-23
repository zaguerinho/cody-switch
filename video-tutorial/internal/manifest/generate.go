package manifest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FormatSourceFiles reads the given file paths and formats them with line
// numbers, suitable for inclusion in a Claude prompt.
func FormatSourceFiles(filePaths []string) (string, error) {
	var parts []string
	for _, fp := range filePaths {
		content, err := os.ReadFile(fp)
		if err != nil {
			return "", fmt.Errorf("source file not found: %s", fp)
		}
		lines := strings.Split(string(content), "\n")
		var numbered []string
		for i, line := range lines {
			numbered = append(numbered, fmt.Sprintf("%4d | %s", i+1, line))
		}
		header := fmt.Sprintf("=== %s (%s) ===", filepath.Base(fp), fp)
		parts = append(parts, header+"\n"+strings.Join(numbered, "\n"))
	}
	return strings.Join(parts, "\n\n"), nil
}

// buildSystemPrompt returns the system prompt that instructs Claude to
// output a structured JSON scene manifest.
func buildSystemPrompt() string {
	return `You are a senior developer recording a screencast tutorial. You produce structured JSON scene manifests for narrated code tutorials.

NARRATION STYLE:
You are teaching, not reading code aloud. Your audience is a developer who wants to understand the WHY, not just the WHAT. Follow these principles:

1. OPEN WITH CONTEXT: Chapter 1 should be a conceptual introduction. Explain what this code IS, what problem it solves, the motivation behind it, and how it fits in the larger project. Point at high-level structures (imports, type definitions, the main entry point) while you explain the big picture.

2. TEACH THE DESIGN: Before diving into implementation details, explain the design decisions. Why was it built this way? What patterns does it use? What are the alternatives and tradeoffs?

3. SHOW HOW IT'S USED: Include a chapter that shows how other code calls or consumes this module. If the code defines an API, class, or function — explain how a consumer would use it, what inputs it expects, what it returns.

4. DON'T STATE THE OBVIOUS: Never say things like "here we define a variable called x" or "this function is called foo". Instead, explain what the variable represents conceptually, why the function exists, and what insight the viewer should take away.

5. ADD INSIGHT: Point out clever patterns, potential pitfalls, edge cases the code handles, and things that might surprise the reader. A good narrator says things the viewer couldn't get from just reading the code.

6. CLOSE WITH PERSPECTIVE: The final chapter should zoom back out — summarize the key takeaways, mention what could be improved or extended, and leave the viewer with a clear mental model.

STRUCTURE:
- 3-6 chapters, each with a clear narrative arc
- Chapter 1: Introduction and motivation (point at imports, types, or the entry point)
- Middle chapters: Design and implementation (teach, don't describe)
- Last chapter: Summary, usage patterns, and what's next
- Each chapter: 60-90 seconds of narration (~150-225 words)
- Each segment: 1-3 sentences

TECHNICAL RULES:
- Output ONLY valid JSON. No markdown fences, no explanation, no preamble.
- Every narration sentence MUST reference only code visible in its paired cue.
- Never reference code in narration that isn't in the current cue's line range.
- Use conversational but precise technical language — like explaining to a colleague.
- Code references must use exact file paths and valid line ranges from the provided sources.
- Line ranges are inclusive: [start, end] means lines start through end.

OUTPUT SCHEMA:
{
  "title": "string — concise, engaging tutorial title",
  "chapters": [
    {
      "id": "ch1",
      "title": "string — chapter title",
      "segments": [
        {
          "id": "ch1-s1",
          "narration": "string — what the narrator says",
          "cue": {
            "type": "highlight",
            "file": "string — relative file path exactly as provided",
            "lines": [startLine, endLine],
            "scroll_to": startLine
          }
        }
      ]
    }
  ]
}`
}

// buildUserPrompt creates the user prompt from a topic description and formatted source code.
func buildUserPrompt(topic, sourceCode string) string {
	return fmt.Sprintf(`Create a narrated code tutorial about:

%s

SOURCE FILES:
%s

Generate a scene manifest JSON with 3-6 chapters. Structure the tutorial like a senior developer explaining this to a new team member:

1. Start with the big picture — what is this, why does it exist, how does it fit in?
2. Explain the design and key decisions before diving into line-by-line details
3. Show how it's used — what calls this code, what does the API look like from outside?
4. Add insight — patterns, tradeoffs, edge cases, things that aren't obvious from reading
5. Close with takeaways — what should the viewer remember, what could be extended?

DO NOT just read the code top to bottom. Teach the concepts using the code as illustration.

Rules:
- Every segment's narration must reference ONLY code in its cue's line range
- Use the exact file paths shown above
- Line numbers must be within the file's actual range
- Co-generate narration and cues together — they must be in sync`, topic, sourceCode)
}

// GatherProjectContext reads project docs from the working directory to give
// Claude richer context for narration. Reads CLAUDE.md, README.md, and
// docs/{feature}/* files. Truncates to maxChars total.
func GatherProjectContext(projectDir string, maxChars int) string {
	var parts []string
	totalChars := 0

	// Priority order: CLAUDE.md, README.md, then docs/ files
	candidates := []string{
		filepath.Join(projectDir, "CLAUDE.md"),
		filepath.Join(projectDir, "README.md"),
	}

	// Add docs/ for the current feature if .claude-current-feature exists
	featureFile := filepath.Join(projectDir, ".claude-current-feature")
	if data, err := os.ReadFile(featureFile); err == nil {
		feature := strings.TrimSpace(string(data))
		if feature != "" {
			docsDir := filepath.Join(projectDir, "docs", feature)
			if entries, err := os.ReadDir(docsDir); err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						candidates = append(candidates, filepath.Join(docsDir, e.Name()))
					}
				}
			}
		}
	}

	// Also scan docs/ root for general project docs
	docsRoot := filepath.Join(projectDir, "docs")
	if entries, err := os.ReadDir(docsRoot); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				fp := filepath.Join(docsRoot, e.Name())
				// Avoid duplicates
				found := false
				for _, c := range candidates {
					if c == fp {
						found = true
						break
					}
				}
				if !found {
					candidates = append(candidates, fp)
				}
			}
		}
	}

	for _, fp := range candidates {
		if totalChars >= maxChars {
			break
		}
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		content := string(data)
		remaining := maxChars - totalChars
		if len(content) > remaining {
			content = content[:remaining] + "\n... (truncated)"
		}
		relPath, _ := filepath.Rel(projectDir, fp)
		if relPath == "" {
			relPath = fp
		}
		parts = append(parts, fmt.Sprintf("=== %s ===\n%s", relPath, content))
		totalChars += len(content)
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

// Generate calls the Claude CLI to produce a scene manifest from a topic and
// source files. It requires the `claude` binary to be on PATH.
// If projectDir is non-empty, project docs are injected as context.
func Generate(ctx context.Context, topic string, filePaths []string, projectDir string) (*Manifest, error) {
	sourceCode, err := FormatSourceFiles(filePaths)
	if err != nil {
		return nil, err
	}

	systemPrompt := buildSystemPrompt()

	// Gather project context if available
	var projectContext string
	if projectDir != "" {
		projectContext = GatherProjectContext(projectDir, 40000)
		if projectContext != "" {
			fmt.Fprintf(os.Stderr, "Injecting project context (%d chars)\n", len(projectContext))
		}
	}

	userPrompt := buildUserPrompt(topic, sourceCode)

	// If we have project context, prepend it
	if projectContext != "" {
		userPrompt = fmt.Sprintf(`PROJECT CONTEXT (use this to understand motivation, architecture, and how this code fits in):

%s

---

%s`, projectContext, userPrompt)
	}

	// Build the full prompt: system instructions + user request
	fullPrompt := systemPrompt + "\n\n---\n\n" + userPrompt

	// Call Claude CLI in print mode
	cmd := exec.CommandContext(ctx, "claude", "-p", fullPrompt, "--output-format", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Fprintf(os.Stderr, "Generating manifest via Claude CLI for: %q\n", truncate(topic, 80))
	fmt.Fprintf(os.Stderr, "Source files: %s\n", formatFileNames(filePaths))

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude CLI failed: %w\nstderr: %s", err, stderr.String())
	}

	// Parse the Claude CLI JSON output
	// The --output-format json wraps the response in {"type":"result","result":"..."}
	var cliOutput struct {
		Result string `json:"result"`
	}
	raw := stdout.Bytes()
	if err := json.Unmarshal(raw, &cliOutput); err != nil {
		// Maybe it returned the manifest directly
		cliOutput.Result = string(raw)
	}

	// Strip any accidental markdown fences
	jsonText := strings.TrimSpace(cliOutput.Result)
	if strings.HasPrefix(jsonText, "```") {
		jsonText = strings.TrimPrefix(jsonText, "```json\n")
		jsonText = strings.TrimPrefix(jsonText, "```\n")
		jsonText = strings.TrimSuffix(jsonText, "\n```")
		jsonText = strings.TrimSuffix(jsonText, "```")
	}

	var manifest Manifest
	if err := json.Unmarshal([]byte(jsonText), &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON:\n%s\n\nparse error: %w",
			truncate(jsonText, 500), err)
	}

	return &manifest, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func formatFileNames(paths []string) string {
	names := make([]string, len(paths))
	for i, p := range paths {
		names[i] = filepath.Base(p)
	}
	return strings.Join(names, ", ")
}
