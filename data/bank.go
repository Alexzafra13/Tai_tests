// Package data embeds the official content into the binary: the syllabus
// of the current call, the question bank (data/bank), the study texts of
// the laws (data/laws) and the study notes (data/notes).
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

//go:embed laws/*.json
var laws embed.FS

//go:embed notes/*.json
var notes embed.FS

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

// Laws returns the study texts (data/laws): one file per law, copied from
// the BOE consolidated legislation, and topics.json linking them to topics.
func Laws() fs.FS {
	sub, err := fs.Sub(laws, "laws")
	if err != nil {
		panic(err)
	}
	return sub
}

// Notes returns the study notes (data/notes), one file per topic.
func Notes() fs.FS {
	sub, err := fs.Sub(notes, "notes")
	if err != nil {
		panic(err)
	}
	return sub
}
