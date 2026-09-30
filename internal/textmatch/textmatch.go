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
