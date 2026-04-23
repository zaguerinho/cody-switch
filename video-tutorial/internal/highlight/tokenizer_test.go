package highlight

import (
	"strings"
	"testing"
)

// render is a convenience helper that tokenizes and renders in one step,
// matching how the JS tokenizeLine function works (returns HTML).
func render(line string) string {
	return RenderTokens(TokenizeLine(line))
}

// TestTokenizeLine ports all 17 tokenizeLine tests from
// video-tutorial/test/unit.js (lines 109–202).
func TestTokenizeLine(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		checks func(t *testing.T, html string)
	}{
		{
			name:  "tokenizes keywords",
			input: "const x = 5;",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-keyword") {
					t.Error("should contain keyword class")
				}
				if !strings.Contains(html, "const") {
					t.Error(`should contain "const"`)
				}
			},
		},
		{
			name:  "tokenizes strings",
			input: `const s = "hello";`,
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-string") {
					t.Error("should contain string class")
				}
				if !strings.Contains(html, "hello") {
					t.Error("should contain string content")
				}
			},
		},
		{
			name:  "tokenizes single-quoted strings",
			input: "const s = 'world';",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-string") {
					t.Error("should contain tk-string")
				}
				if !strings.Contains(html, "world") {
					t.Error("should contain world")
				}
			},
		},
		{
			name:  "tokenizes numbers",
			input: "const n = 42;",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-number") {
					t.Error("should contain tk-number")
				}
				if !strings.Contains(html, "42") {
					t.Error("should contain 42")
				}
			},
		},
		{
			name:  "tokenizes hex numbers",
			input: "const mask = 0xFF;",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-number") {
					t.Error("should contain tk-number")
				}
				if !strings.Contains(html, "0xFF") {
					t.Error("should contain 0xFF")
				}
			},
		},
		{
			name:  "tokenizes function calls",
			input: `console.log("hi");`,
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-function") {
					t.Error("should contain tk-function")
				}
				if !strings.Contains(html, "log") {
					t.Error("should contain log")
				}
			},
		},
		{
			name:  "tokenizes line comments //",
			input: "// comment here",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-comment") {
					t.Error("should contain tk-comment")
				}
				if !strings.Contains(html, "comment here") {
					t.Error("should contain comment here")
				}
			},
		},
		{
			name:  "tokenizes Python # comments",
			input: "  # Python comment",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-comment") {
					t.Error("should contain tk-comment")
				}
			},
		},
		{
			name:  "tokenizes block comment opening",
			input: "/* block comment */",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-comment") {
					t.Error("should contain tk-comment")
				}
			},
		},
		{
			name:  "tokenizes template literals with interpolation",
			input: "const msg = `hello ${name}`;",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-string") {
					t.Error("should contain tk-string")
				}
				if !strings.Contains(html, "hello") {
					t.Error("should contain hello")
				}
				if !strings.Contains(html, "name") {
					t.Error("should contain name")
				}
			},
		},
		{
			name:  "tokenizes triple-quoted strings",
			input: `doc = """docstring"""`,
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-string") {
					t.Error("should contain tk-string")
				}
				if !strings.Contains(html, "docstring") {
					t.Error("should contain docstring")
				}
			},
		},
		{
			name:  "tokenizes decorators",
			input: `@app.route("/api")`,
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-function") {
					t.Error("should contain tk-function")
				}
				if !strings.Contains(html, "@app.route") {
					t.Error("should contain @app.route")
				}
			},
		},
		{
			name:  "escapes HTML entities",
			input: "if (a < b && c > d) {}",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "&lt;") {
					t.Error("should escape <")
				}
				if !strings.Contains(html, "&gt;") {
					t.Error("should escape >")
				}
				if !strings.Contains(html, "&amp;") {
					t.Error("should escape &")
				}
			},
		},
		{
			name:  "handles empty line",
			input: "",
			checks: func(t *testing.T, html string) {
				if html != "" {
					t.Errorf("expected empty string, got %q", html)
				}
			},
		},
		{
			name:  "handles whitespace-only line",
			input: "    ",
			checks: func(t *testing.T, html string) {
				if html != "    " {
					t.Errorf("expected %q, got %q", "    ", html)
				}
			},
		},
		{
			name:  "inline comment after code",
			input: "const x = 1; // set x",
			checks: func(t *testing.T, html string) {
				if !strings.Contains(html, "tk-keyword") {
					t.Error("should have keyword")
				}
				if !strings.Contains(html, "tk-number") {
					t.Error("should have number")
				}
				if !strings.Contains(html, "tk-comment") {
					t.Error("should have comment")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			html := render(tc.input)
			tc.checks(t, html)
		})
	}
}

// TestTokenizeLineTokenTypes verifies the raw token types produced by
// TokenizeLine (not the HTML rendering). This is the 17th test
// equivalent — a structural sanity check on the token slice.
func TestTokenizeLineTokenTypes(t *testing.T) {
	t.Run("const x = 5 produces keyword, plain, number tokens", func(t *testing.T) {
		tokens := TokenizeLine("const x = 5;")
		if len(tokens) == 0 {
			t.Fatal("expected tokens")
		}
		if tokens[0].Type != TokenKeyword || tokens[0].Text != "const" {
			t.Errorf("first token: got %v %q, want keyword 'const'", tokens[0].Type, tokens[0].Text)
		}
		// Find the number token.
		found := false
		for _, tok := range tokens {
			if tok.Type == TokenNumber && tok.Text == "5" {
				found = true
			}
		}
		if !found {
			t.Error("expected a number token with text '5'")
		}
	})
}

// TestRenderTokens verifies the HTML rendering logic independently.
func TestRenderTokens(t *testing.T) {
	t.Run("plain tokens have no span wrapper", func(t *testing.T) {
		html := RenderTokens([]Token{{Type: TokenPlain, Text: "hello"}})
		if html != "hello" {
			t.Errorf("got %q, want %q", html, "hello")
		}
	})

	t.Run("keyword tokens get tk-keyword class", func(t *testing.T) {
		html := RenderTokens([]Token{{Type: TokenKeyword, Text: "const"}})
		want := `<span class="tk-keyword">const</span>`
		if html != want {
			t.Errorf("got %q, want %q", html, want)
		}
	})

	t.Run("HTML entities are escaped", func(t *testing.T) {
		html := RenderTokens([]Token{{Type: TokenPlain, Text: "a < b & c > d"}})
		if !strings.Contains(html, "&lt;") {
			t.Error("< not escaped")
		}
		if !strings.Contains(html, "&amp;") {
			t.Error("& not escaped")
		}
		if !strings.Contains(html, "&gt;") {
			t.Error("> not escaped")
		}
	})

	t.Run("decorator tokens render with tk-function class", func(t *testing.T) {
		html := RenderTokens([]Token{{Type: TokenDecorator, Text: "@route"}})
		want := `<span class="tk-function">@route</span>`
		if html != want {
			t.Errorf("got %q, want %q", html, want)
		}
	})
}
