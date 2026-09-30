// Package textmatch checks that quoted fragments appear literally in a source
// text. It is the guard behind the "nothing made up" rule: a question's
// source_quote must be found in its source after normalizing only
// presentation differences (whitespace and typographic punctuation), never
// wording or case.
package textmatch

import (
	"strings"
	"unicode"
)

var replacer = strings.NewReplacer(
	"“", `"`, "”", `"`, "«", `"`, "»", `"`, "„", `"`,
	"‘", "'", "’", "'", "‚", "'",
	"–", "-", "—", "-", "‑", "-", "−", "-",
	"…", "...",
	"\u00ad", "", // soft hyphen
	"\u200b", "", // zero-width space
	"\ufeff", "", // BOM
)

// Normalize collapses every run of whitespace (including non-breaking
// spaces and newlines) to a single space, unifies typographic quotes,
// dashes and ellipses, and trims the result.
func Normalize(s string) string {
	s = replacer.Replace(s)
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}

// Contains reports whether quote appears in text after normalizing both.
// An empty quote never matches.
func Contains(text, quote string) bool {
	q := Normalize(quote)
	if q == "" {
		return false
	}
	return strings.Contains(Normalize(text), q)
}

// Excerpt is a quote located in its source text, with surrounding context,
// all in normalized form.
type Excerpt struct {
	Before string `json:"before"`
	Match  string `json:"match"`
	After  string `json:"after"`
	// Clipped tells whether the context was cut at either end.
	ClippedStart bool `json:"clipped_start"`
	ClippedEnd   bool `json:"clipped_end"`
}

// Locate finds quote in text and returns it with up to window characters of
// context on each side, cut at word boundaries. Before+Match+After reads as
// the original passage. It reports false if the quote is not in the text.
func Locate(text, quote string, window int) (Excerpt, bool) {
	t, q := []rune(Normalize(text)), Normalize(quote)
	if q == "" {
		return Excerpt{}, false
	}
	// Index on runes so the window never splits a character.
	byteIdx := strings.Index(string(t), q)
	if byteIdx < 0 {
		return Excerpt{}, false
	}
	start := len([]rune(string(t)[:byteIdx]))
	end := start + len([]rune(q))

	from := max(0, start-window)
	for from > 0 && from < start && t[from-1] != ' ' {
		from++
	}
	to := min(len(t), end+window)
	for to < len(t) && to > end && t[to] != ' ' {
		to--
	}
	return Excerpt{
		Before:       strings.TrimLeft(string(t[from:start]), " "),
		Match:        string(t[start:end]),
		After:        strings.TrimRight(string(t[end:to]), " "),
		ClippedStart: from > 0,
		ClippedEnd:   to < len(t),
	}, true
}
