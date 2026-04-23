// Package embed provides embedded browser assets and templates for the
// video-tutorial binary. These files are baked into the binary at compile
// time and inlined into the generated HTML tutorial.
package embed

import "embed"

//go:embed browser/sync.js browser/renderer.js browser/chatbot.js template/player.css
var Assets embed.FS

// ReadAsset reads an embedded file and returns its contents.
func ReadAsset(path string) ([]byte, error) {
	return Assets.ReadFile(path)
}

// MustReadAsset reads an embedded file and panics on error.
func MustReadAsset(path string) string {
	data, err := Assets.ReadFile(path)
	if err != nil {
		panic("embedded asset missing: " + path + ": " + err.Error())
	}
	return string(data)
}
