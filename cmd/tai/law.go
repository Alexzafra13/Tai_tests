package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	ref := fs.String("ref", "", "BOE reference of the law, e.g. BOE-A-2015-10565, or CELEX number of an EU regulation, e.g. 32016R0679")
	alias := fs.String("alias", "", `extra names questions use for it, comma separated, e.g. "Constitución Española,CE"`)
	dir := fs.String("dir", "data/laws", "directory of the law files")
	fs.Parse(args)
	if *ref == "" {
		fs.Usage()
		return fmt.Errorf("-ref is required")
	}
	_, err := fetchLaw(ctx, log, *ref, strings.Split(*alias, ","), *dir)
	return err
}

// runRefreshLaws downloads again every law of data/laws, keeping the
// names given with -alias, so the bundled texts follow the BOE. A
// scheduled workflow runs it and proposes the changes.
func runRefreshLaws(ctx context.Context, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("refresh-laws", flag.ExitOnError)
	dir := fs.String("dir", "data/laws", "directory of the law files")
	fs.Parse(args)
	names, err := filepath.Glob(filepath.Join(*dir, "*.json"))
	if err != nil {
		return err
	}
	changed := 0
	var repealed []string
	for _, name := range names {
		if filepath.Base(name) == "topics.json" {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		var old content.LawFile
		if err := json.Unmarshal(b, &old); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		derived := map[string]bool{}
		for _, a := range lawtext.Aliases(old.Title, nil) {
			derived[a] = true
		}
		var extra []string
		for _, a := range old.Aliases {
			if !derived[a] {
				extra = append(extra, a)
			}
		}
		updated, err := fetchLaw(ctx, log, old.Reference, extra, *dir)
		var rep repealedError
		if errors.As(err, &rep) {
			repealed = append(repealed, rep.ref+" on "+rep.date)
			continue
		}
		if err != nil {
			return err
		}
		if !bytes.Equal(updated, b) {
			changed++
		}
	}
	log.Info("laws refreshed", "laws", len(names)-1, "changed", changed)
	if len(repealed) > 0 {
		// Failing makes the scheduled run red, so someone replaces them.
		return fmt.Errorf("repealed laws, replace them with the ones in force: %s", strings.Join(repealed, "; "))
	}
	return nil
}

// repealedError reports a BOE law that is no longer in force: its text
// must not be offered as the one to study.
type repealedError struct {
	ref, date string
}

func (e repealedError) Error() string { return "repealed on " + e.date }

// fetchLaw writes the law's file in dir and returns its content. ref is a
// BOE reference or, for an EU regulation, its CELEX number.
func fetchLaw(ctx context.Context, log *slog.Logger, ref string, extraAliases []string, dir string) ([]byte, error) {
	var law content.LawFile
	var err error
	if lawtext.IsCELEX(ref) {
		law, err = fetchEURLex(ctx, ref)
	} else {
		law, err = fetchBOE(ctx, ref)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ref, err)
	}
	law.Aliases = lawtext.Aliases(law.Title, extraAliases)
	sections := law.Sections
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(law); err != nil {
		return nil, err
	}
	out := filepath.Join(dir, ref+".json")
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return nil, err
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
	log.Info("law written", "file", out, "title", law.Title, "version", law.VersionDate,
		"sections", len(sections), "articles", articles, "changes_pending", upcoming)
	return buf.Bytes(), nil
}

func fetchBOE(ctx context.Context, ref string) (content.LawFile, error) {
	metaJSON, err := httpGet(ctx, lawtext.MetadataURL(ref), "application/json")
	if err != nil {
		return content.LawFile{}, err
	}
	meta, err := lawtext.ParseMetadata(metaJSON)
	if err != nil {
		return content.LawFile{}, err
	}
	if meta.Repealed != "" {
		return content.LawFile{}, repealedError{ref: ref, date: meta.Repealed}
	}
	textXML, err := httpGet(ctx, lawtext.TextURL(ref), "application/xml")
	if err != nil {
		return content.LawFile{}, err
	}
	sections, err := lawtext.ParseText(textXML, time.Now())
	if err != nil {
		return content.LawFile{}, err
	}
	return content.LawFile{Reference: ref, Title: meta.Title, URL: lawtext.PageURL(ref), VersionDate: meta.VersionDate,
		Sections: sections}, nil
}

// fetchEURLex takes the latest consolidated version of an EU regulation.
// EUR-Lex has no future wordings: changes appear once in force.
func fetchEURLex(ctx context.Context, celex string) (content.LawFile, error) {
	page, err := httpGet(ctx, lawtext.EURLexVersionsURL(celex), "text/html")
	if err != nil {
		return content.LawFile{}, err
	}
	version, err := lawtext.LatestEURLexVersion(page, celex)
	if err != nil {
		return content.LawFile{}, err
	}
	day, err := lawtext.VersionDay(version)
	if err != nil {
		return content.LawFile{}, err
	}
	text, err := httpGet(ctx, lawtext.EURLexTextURL(celex, version), "text/html")
	if err != nil {
		return content.LawFile{}, err
	}
	title, sections, err := lawtext.ParseEURLexText(text)
	if err != nil {
		return content.LawFile{}, err
	}
	return content.LawFile{Reference: celex, Title: title, URL: lawtext.EURLexPageURL(celex), VersionDate: day,
		Sections: sections}, nil
}

// httpGet retries a 202 answer: EUR-Lex sends it, with no content, while
// its bot protection holds requests back.
func httpGet(ctx context.Context, url, accept string) ([]byte, error) {
	for wait := 5 * time.Second; ; wait *= 2 {
		b, status, err := httpGetOnce(ctx, url, accept)
		if err != nil {
			return nil, err
		}
		switch {
		case status == http.StatusOK:
			return b, nil
		case status == http.StatusAccepted && wait <= time.Minute:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		default:
			return nil, fmt.Errorf("GET %s: %d %s", url, status, http.StatusText(status))
		}
	}
}

func httpGetOnce(ctx context.Context, url, accept string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", accept)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return b, resp.StatusCode, nil
}
