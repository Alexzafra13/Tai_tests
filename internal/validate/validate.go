// Package validate holds the error type used for invalid user input across
// packages: field names mapped to human-readable (Spanish) messages that
// forms show next to each field.
package validate

import (
	"sort"
	"strings"
)

type Errors map[string]string

func (v Errors) Error() string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + v[k]
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// Err returns v as an error, or nil when there are no errors.
func (v Errors) Err() error {
	if len(v) == 0 {
		return nil
	}
	return v
}
