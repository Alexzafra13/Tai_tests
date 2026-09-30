package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/alexzafra13/tai_tests/internal/content"
)

// openInput opens path for reading; "-" means standard input, which lets
// Docker users pipe files in: docker compose exec -T tai tai load-syllabus -file - < data/syllabus.json
func openInput(path string) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(os.Stdin), nil
	}
	return os.Open(path)
}

func runLoadSyllabus(ctx context.Context, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("load-syllabus", flag.ExitOnError)
	file := fs.String("file", "data/syllabus.json", `syllabus JSON file ("-" for stdin)`)
	fs.Parse(args)

	r, err := openInput(*file)
	if err != nil {
		return err
	}
	defer r.Close()
	f, err := content.ParseSyllabus(r)
	if err != nil {
		return err
	}

	d, err := openDB(ctx, log)
	if err != nil {
		return err
	}
	defer d.Close()
	res, err := content.NewStore(d).LoadSyllabus(ctx, f)
	if err != nil {
		return err
	}
	log.Info("syllabus loaded", "name", f.Name, "blocks", res.Blocks, "topics", res.Topics, "inactive_topics", res.Deactivated)
	return nil
}

func runAddSource(ctx context.Context, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("add-source", flag.ExitOnError)
	kind := fs.String("kind", "", "source kind: law, technical_doc or inap_exam")
	title := fs.String("title", "", "title, e.g. \"Ley 39/2015, del Procedimiento Administrativo Común\"")
	ref := fs.String("ref", "", "reference, e.g. BOE-A-2015-10565")
	url := fs.String("url", "", "URL of the original document")
	version := fs.String("version", "", "version date of the text (YYYY-MM-DD)")
	textFile := fs.String("text", "", `plain-text file with the full text ("-" for stdin)`)
	fs.Parse(args)

	in := content.SourceInput{Kind: content.SourceKind(*kind), Title: *title, Reference: *ref, URL: *url, VersionDate: *version}
	if *textFile != "" {
		r, err := openInput(*textFile)
		if err != nil {
			return err
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			return err
		}
		in.FullText = string(b)
	}

	d, err := openDB(ctx, log)
	if err != nil {
		return err
	}
	defer d.Close()
	id, err := content.NewStore(d).CreateSource(ctx, in)
	var v content.ValidationError
	if errors.As(err, &v) {
		for field, msg := range v {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", field, msg)
		}
	}
	if err != nil {
		return err
	}
	log.Info("source added", "id", id, "kind", in.Kind, "title", in.Title, "chars", len(in.FullText))
	return nil
}
