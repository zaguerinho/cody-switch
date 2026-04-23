package fileutil

import (
	"path/filepath"
	"strings"
)

// codeExtensions maps file extensions (without the dot) to true for all
// recognized code file types. Keep in sync with DetectLanguageExtensions.
var codeExtensions = map[string]bool{
	"py":   true,
	"js":   true,
	"ts":   true,
	"jsx":  true,
	"tsx":  true,
	"go":   true,
	"rb":   true,
	"php":  true,
	"java": true,
	"rs":   true,
	"c":    true,
	"cpp":  true,
	"h":    true,
}

// DetectLanguageExtensions returns the default recursive glob pattern that
// matches all recognized code file types. The pattern uses brace expansion
// compatible with doublestar.
func DetectLanguageExtensions() string {
	return "**/*.{py,js,ts,jsx,tsx,go,rb,php,java,rs,c,cpp,h}"
}

// IsCodeFile reports whether path has a file extension recognized as a code file.
func IsCodeFile(path string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	return codeExtensions[strings.ToLower(ext)]
}
