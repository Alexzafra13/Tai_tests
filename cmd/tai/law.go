package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/content/lawtext"
)

// runFetchLaw downloads the consolidated text of a law from the BOE open
// data API into a file of data/laws. Installations get it on their next
// start.
func runFetchLaw(ctx context.Context, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("fetch-law", flag.ExitOnError)
	ref := fs.String("ref", "", "BOE reference of the law, e.g. BOE-A-2015-10565")
	alias := fs.String("alias", "", `extra names questions use for it, comma separated, e.g. "Constitución Española,CE"`)
	dir := fs.String("dir", "data/laws", "directory of the law files")
	fs.Parse(args)
	if *ref == "" {
		fs.Usage()
		return fmt.Errorf("-ref is required")
	}

	metaJSON, err := httpGet(ctx, lawtext.MetadataURL(*ref), "application/json")
	if err != nil {
		return err
	}
	meta, err := lawtext.ParseMetadata(metaJSON)
	if err != nil {
		return fmt.Errorf("%s: %w", *ref, err)
	}
	textXML, err := httpGet(ctx, lawtext.TextURL(*ref), "application/xml")
	if err != nil {
		return err
	}
	sections, err := lawtext.ParseText(textXML, time.Now())
	if err != nil {
		return fmt.Errorf("%s: %w", *ref, err)
	}

	law := content.LawFile{
		Reference: *ref, Title: meta.Title, URL: lawtext.PageURL(*ref), VersionDate: meta.VersionDate,
		Aliases: lawtext.Aliases(meta.Title, strings.Split(*alias, ",")), Sections: sections,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(law); err != nil {
		return err
	}
	out := filepath.Join(*dir, *ref+".json")
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return err
	}

	articles, upcoming := 0, 0
	for _, s := range sections {
		if s.Kind == content.SectionArticle {
			articles++
		}
		if s.Upcoming != nil {
			upcoming++
		}
	}
	log.Info("law written", "file", out, "title", meta.Title, "version", meta.VersionDate,
		"sections", len(sections), "articles", articles, "changes_pending", upcoming)
	return nil
}

func httpGet(ctx context.Context, url, accept string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return b, nil
}
