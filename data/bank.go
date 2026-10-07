// Package data embeds the question bank (data/bank) into the binary.
package data

import (
	"embed"
	"io/fs"
)

//go:embed bank/*.json
var bank embed.FS

// Bank returns the bank files, rooted at data/bank.
func Bank() fs.FS {
	sub, err := fs.Sub(bank, "bank")
	if err != nil {
		panic(err)
	}
	return sub
}
