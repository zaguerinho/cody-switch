package assembler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zaguerinho/claude-switch/video-tutorial/internal/highlight"
	"github.com/zaguerinho/claude-switch/video-tutorial/internal/manifest"
)

// BuildCodeViewer builds the HTML for file tabs and syntax-highlighted code blocks.
// It returns the file tabs HTML and code blocks HTML as separate strings.
func BuildCodeViewer(m *manifest.Manifest, sourceDir string) (fileTabsHTML, codeBlocksHTML string, err error) {
	// Collect unique files from manifest cues in order of first appearance.
	seen := make(map[string]bool)
	var fileList []string
	for _, ch := range m.Chapters {
		for _, seg := range ch.Segments {
			if seg.Cue.File != "" && !seen[seg.Cue.File] {
				seen[seg.Cue.File] = true
				fileList = append(fileList, seg.Cue.File)
			}
		}
	}

	// Build file tabs.
	var tabsBuilder strings.Builder
	for i, f := range fileList {
		active := ""
		if i == 0 {
			active = " file-tab--active"
		}
		name := filepath.Base(f)
		fmt.Fprintf(&tabsBuilder, `      <button class="file-tab%s" data-file="%s">
        <svg class="file-tab-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
        </svg>
        %s
      </button>`, active, f, name)
		if i < len(fileList)-1 {
			tabsBuilder.WriteByte('\n')
		}
	}

	// Build code blocks for each file.
	var blocksBuilder strings.Builder
	for fileIdx, f := range fileList {
		filePath := filepath.Join(sourceDir, f)
		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return "", "", fmt.Errorf("assembler: source file not found: %s (referenced as %q)", filePath, f)
		}
		content := string(data)
		lines := strings.Split(content, "\n")

		display := ""
		if fileIdx > 0 {
			display = ` style="display:none"`
		}

		// Build line HTML.
		var linesBuilder strings.Builder
		for lineIdx, line := range lines {
			num := lineIdx + 1
			tokens := highlight.TokenizeLine(line)
			rendered := highlight.RenderTokens(tokens)
			fmt.Fprintf(&linesBuilder, `      <div class="code-line" data-line="%d">
        <span class="line-number">%d</span>
        <span class="line-content">%s</span>
      </div>`, num, num, rendered)
			if lineIdx < len(lines)-1 {
				linesBuilder.WriteByte('\n')
			}
		}

		fmt.Fprintf(&blocksBuilder, `    <div class="code-block" data-file="%s"%s>
%s
    </div>`, f, display, linesBuilder.String())
		if fileIdx < len(fileList)-1 {
			blocksBuilder.WriteByte('\n')
		}
	}

	return tabsBuilder.String(), blocksBuilder.String(), nil
}

// BuildChapterList builds the sidebar chapter list HTML.
func BuildChapterList(m *manifest.Manifest) string {
	var b strings.Builder
	for i, ch := range m.Chapters {
		fmt.Fprintf(&b, `      <li class="chapter-item" data-chapter="%s">
        <span class="chapter-status">
          <span class="status-dot"></span>
        </span>
        <div class="chapter-info">
          <div class="chapter-title">%s</div>
        </div>
      </li>`, ch.ID, ch.Title)
		if i < len(m.Chapters)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
