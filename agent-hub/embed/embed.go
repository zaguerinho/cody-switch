package embed

import "embed"

//go:embed templates/index.html templates/PROTOCOL.md templates/MANIFESTO.md templates/STATUS.md templates/PLAYBOOK.md static/style.css static/app.js
var Assets embed.FS

// ReadAsset reads an embedded file.
func ReadAsset(path string) ([]byte, error) {
	return Assets.ReadFile(path)
}

// GovernanceTemplates returns the default governance doc templates.
func GovernanceTemplates() map[string][]byte {
	docs := map[string][]byte{}
	for _, name := range []string{"PROTOCOL.md", "MANIFESTO.md", "STATUS.md", "PLAYBOOK.md"} {
		data, err := Assets.ReadFile("templates/" + name)
		if err == nil {
			docs[name] = data
		}
	}
	return docs
}
