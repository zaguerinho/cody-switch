// Package highlight provides syntax tokenization and HTML rendering for
// common programming languages. It is a Go port of the tokenizeLine /
// renderTokens functions from the video-tutorial JS assembler.
package highlight

import (
	"unicode"
)

// TokenType classifies a syntax token.
type TokenType string

const (
	TokenKeyword     TokenType = "keyword"
	TokenString      TokenType = "string"
	TokenComment     TokenType = "comment"
	TokenNumber      TokenType = "number"
	TokenFunction    TokenType = "function"
	TokenType_       TokenType = "type"
	TokenDecorator   TokenType = "decorator"
	TokenOperator    TokenType = "operator"
	TokenPunctuation TokenType = "punctuation"
	TokenPlain       TokenType = "plain"
)

// Token is a single syntax element produced by the tokenizer.
type Token struct {
	Type TokenType
	Text string
}

// keywords is the set of language keywords recognised by the tokenizer.
// It covers JS/TS, Python, Go, Ruby, PHP, Java, and C.
var keywords = map[string]bool{
	"const": true, "let": true, "var": true, "function": true, "return": true,
	"if": true, "else": true, "for": true, "while": true, "do": true,
	"switch": true, "case": true, "break": true, "continue": true,
	"class": true, "import": true, "export": true, "from": true,
	"default": true, "async": true, "await": true, "try": true, "catch": true,
	"finally": true, "throw": true, "new": true, "typeof": true,
	"instanceof": true, "in": true, "of": true, "true": true, "false": true,
	"null": true, "undefined": true, "this": true,
	"def": true, "self": true, "elif": true, "pass": true, "raise": true,
	"with": true, "as": true, "yield": true, "lambda": true,
	"func": true, "go": true, "defer": true, "chan": true, "select": true,
	"range": true, "package": true, "type": true, "struct": true,
	"interface": true, "nil": true, "None": true, "True": true, "False": true,
	"not": true, "and": true, "or": true, "is": true, "print": true,
	"extends": true, "implements": true, "static": true, "public": true,
	"private": true, "protected": true, "final": true,
	"abstract": true, "super": true, "void": true, "int": true, "float": true,
	"double": true, "bool": true, "string": true, "char": true,
}

// TokenizeLine splits a single source line into classified tokens.
// It matches the behaviour of the JavaScript tokenizeLine in assemble.js.
func TokenizeLine(line string) []Token {
	tokens := []Token{}
	runes := []rune(line)
	n := len(runes)
	i := 0
	var buf []rune

	flushBuf := func() {
		if len(buf) > 0 {
			tokens = append(tokens, Token{Type: TokenPlain, Text: string(buf)})
			buf = buf[:0]
		}
	}

	for i < n {
		ch := runes[i]

		// ── Line comments: // ──────────────────────────────────────────
		if ch == '/' && i+1 < n && runes[i+1] == '/' {
			flushBuf()
			tokens = append(tokens, Token{Type: TokenComment, Text: string(runes[i:])})
			i = n
			continue
		}

		// ── Python/Ruby/Shell # comments (only at start or after whitespace) ──
		if ch == '#' && (i == 0 || isWhitespace(runes[i-1])) {
			flushBuf()
			tokens = append(tokens, Token{Type: TokenComment, Text: string(runes[i:])})
			i = n
			continue
		}

		// ── Block comment opening: /* ... ─────────────────────────────
		if ch == '/' && i+1 < n && runes[i+1] == '*' {
			flushBuf()
			tokens = append(tokens, Token{Type: TokenComment, Text: string(runes[i:])})
			i = n
			continue
		}

		// ── Block comment continuation: lines starting with * ────────
		if ch == '*' {
			trimmed := trimLeftRunes(runes)
			if len(trimmed) > 0 && trimmed[0] == '*' {
				// Check that i is at the position of the first * in the trimmed line.
				leadingSpaces := n - len(trimmed)
				if i == leadingSpaces {
					flushBuf()
					tokens = append(tokens, Token{Type: TokenComment, Text: string(runes[i:])})
					i = n
					continue
				}
			}
		}

		// ── Template literals with ${} interpolation ─────────────────
		if ch == '`' {
			flushBuf()
			j := i + 1
			depth := 0
			for j < n {
				if runes[j] == '\\' {
					j += 2
					continue
				}
				if runes[j] == '$' && j+1 < n && runes[j+1] == '{' {
					depth++
					j += 2
					continue
				}
				if runes[j] == '}' && depth > 0 {
					depth--
					j++
					continue
				}
				if runes[j] == '`' && depth == 0 {
					j++
					break
				}
				j++
			}
			tokens = append(tokens, Token{Type: TokenString, Text: string(runes[i:j])})
			i = j
			continue
		}

		// ── String literals (single and double quotes) ───────────────
		if ch == '"' || ch == '\'' {
			flushBuf()
			quote := ch
			// Check for triple-quoted strings (Python).
			if i+2 < n && runes[i+1] == quote && runes[i+2] == quote {
				j := i + 3
				// Search for closing triple quote.
				endIdx := indexTriple(runes, j, quote)
				if endIdx >= 0 {
					j = endIdx + 3
				} else {
					j = n
				}
				tokens = append(tokens, Token{Type: TokenString, Text: string(runes[i:j])})
				i = j
				continue
			}
			j := i + 1
			for j < n && runes[j] != quote {
				if runes[j] == '\\' {
					j++
				}
				j++
			}
			if j+1 <= n {
				j++
			}
			if j > n {
				j = n
			}
			tokens = append(tokens, Token{Type: TokenString, Text: string(runes[i:j])})
			i = j
			continue
		}

		// ── Numbers (decimal, hex, binary, octal) ────────────────────
		if isDigit(ch) && (i == 0 || !isWordChar(runes[i-1])) {
			flushBuf()
			j := i
			if ch == '0' && j+1 < n && (runes[j+1] == 'x' || runes[j+1] == 'X') {
				j += 2
				for j < n && isHexDigitOrUnderscore(runes[j]) {
					j++
				}
			} else if ch == '0' && j+1 < n && (runes[j+1] == 'b' || runes[j+1] == 'B') {
				j += 2
				for j < n && (runes[j] == '0' || runes[j] == '1' || runes[j] == '_') {
					j++
				}
			} else if ch == '0' && j+1 < n && (runes[j+1] == 'o' || runes[j+1] == 'O') {
				j += 2
				for j < n && isOctalDigitOrUnderscore(runes[j]) {
					j++
				}
			} else {
				for j < n && isDecimalPart(runes[j]) {
					j++
				}
			}
			tokens = append(tokens, Token{Type: TokenNumber, Text: string(runes[i:j])})
			i = j
			continue
		}

		// ── Decorators (@something) ──────────────────────────────────
		if ch == '@' && i+1 < n && isIdentStart(runes[i+1]) {
			flushBuf()
			j := i + 1
			for j < n && isWordOrDot(runes[j]) {
				j++
			}
			tokens = append(tokens, Token{Type: TokenDecorator, Text: string(runes[i:j])})
			i = j
			continue
		}

		// ── Identifiers (keywords and function names) ────────────────
		if isIdentStart(ch) {
			flushBuf()
			j := i
			for j < n && isWordChar(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			// Check if followed by ( → function call (skip spaces).
			afterWord := j
			for afterWord < n && runes[afterWord] == ' ' {
				afterWord++
			}
			if afterWord < n && runes[afterWord] == '(' {
				tokens = append(tokens, Token{Type: TokenFunction, Text: word})
			} else if keywords[word] {
				tokens = append(tokens, Token{Type: TokenKeyword, Text: word})
			} else {
				tokens = append(tokens, Token{Type: TokenPlain, Text: word})
			}
			i = j
			continue
		}

		// ── Everything else accumulates into the plain-text buffer ───
		buf = append(buf, ch)
		i++
	}

	flushBuf()
	return tokens
}

// ── helper predicates ────────────────────────────────────────────────

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isWordChar(r rune) bool {
	// Matches JS \w: [a-zA-Z0-9_]
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func isIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isHexDigitOrUnderscore(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') || r == '_'
}

func isOctalDigitOrUnderscore(r rune) bool {
	return (r >= '0' && r <= '7') || r == '_'
}

func isDecimalPart(r rune) bool {
	// Matches JS [\d._eE]
	return (r >= '0' && r <= '9') || r == '.' || r == '_' || r == 'e' || r == 'E'
}

func isWordOrDot(r rune) bool {
	return isWordChar(r) || r == '.'
}

// trimLeftRunes returns the rune slice with leading whitespace removed.
func trimLeftRunes(rs []rune) []rune {
	for i, r := range rs {
		if !unicode.IsSpace(r) {
			return rs[i:]
		}
	}
	return nil
}

// indexTriple finds the index of three consecutive quote runes starting from pos.
// Returns -1 if not found.
func indexTriple(runes []rune, start int, quote rune) int {
	for i := start; i+2 < len(runes); i++ {
		if runes[i] == quote && runes[i+1] == quote && runes[i+2] == quote {
			return i
		}
	}
	return -1
}
