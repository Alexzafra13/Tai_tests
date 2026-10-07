// Package data embeds the official content into the binary: the syllabus
// of the current call and the question bank (data/bank).
package data

import (
	"embed"
	"io"
	"io/fs"
	"strings"
)

//go:embed bank/*.json
var bank embed.FS

//go:embed syllabus.json
var syllabus string

// Syllabus returns data/syllabus.json, the syllabus of the current call
// copied literally from the BOE.
func Syllabus() io.Reader { return strings.NewReader(syllabus) }

// Bank returns the bank files, rooted at data/bank.
func Bank() fs.FS {
	sub, err := fs.Sub(bank, "bank")
	if err != nil {
		panic(err)
	}
	return sub
}
