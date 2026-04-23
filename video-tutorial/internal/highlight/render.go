package highlight

import "strings"

// cssClassMap maps token types to their CSS class names used in the
// generated HTML. This matches the original JavaScript renderTokens in
// assemble.js. Decorators are rendered with the "tk-function" class to
// match the JS behaviour (the JS tokenizer classifies decorators as
// function tokens).
var cssClassMap = map[TokenType]string{
	TokenComment:     "tk-comment",
	TokenString:      "tk-string",
	TokenKeyword:     "tk-keyword",
	TokenFunction:    "tk-function",
	TokenNumber:      "tk-number",
	TokenType_:       "tk-type",
	TokenDecorator:   "tk-function", // matches original JS behaviour
	TokenOperator:    "tk-operator",
	TokenPunctuation: "tk-punct",
}

// RenderTokens converts a slice of tokens into an HTML string with
// <span class="tk-*"> wrappers. HTML entities in the token text are
// escaped before wrapping. Plain tokens are emitted as escaped text
// without a span wrapper.
func RenderTokens(tokens []Token) string {
	var b strings.Builder
	for _, t := range tokens {
		escaped := escapeHTML(t.Text)
		cls, ok := cssClassMap[t.Type]
		if ok {
			b.WriteString(`<span class="`)
			b.WriteString(cls)
			b.WriteString(`">`)
			b.WriteString(escaped)
			b.WriteString(`</span>`)
		} else {
			// TokenPlain or any unmapped type — emit raw escaped text.
			b.WriteString(escaped)
		}
	}
	return b.String()
}

// escapeHTML replaces &, <, >, and " with their HTML entity equivalents.
func escapeHTML(s string) string {
	// Order matters: & must be replaced first.
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
