package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/content/examtext"
)

// runImportExam turns an INAP exam (question booklet and answer keys, as
// PDF) into a file of the bundled question bank. The questions reach an
// installation as drafts the next time it starts.
func runImportExam(ctx context.Context, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("import-exam", flag.ExitOnError)
	title := fs.String("title", "", `source title, e.g. "TAI ingreso libre · OEP 2024 · ejercicio único (modelo A)"`)
	ref := fs.String("ref", "", "source reference, stable, e.g. INAP-TAI-L-OEP2024")
	url := fs.String("url", "", "page of the selection process in the INAP site")
	label := fs.String("label", "", `start of each question reference, e.g. "OEP 2024"`)
	questions := fs.String("questions", "", "question booklet PDF")
	answers := fs.String("answers", "", "definitive answer key PDF (never the provisional one)")
	provisional := fs.String("provisional", "", "provisional answer key PDF, for the answer of annulled questions")
	out := fs.String("out", "", "bank file to write, e.g. data/bank/inap-tai-l-oep2024.json")
	fs.Parse(args)
	if *title == "" || *ref == "" || *label == "" || *questions == "" || *answers == "" || *out == "" {
		fs.Usage()
		return errors.New("-title, -ref, -label, -questions, -answers and -out are required")
	}

	text, err := pdfText(ctx, *questions)
	if err != nil {
		return err
	}
	booklet, problems := examtext.ParseBooklet(text)
	exam := examtext.Exam{Booklet: booklet, Label: *label}
	if exam.Final, err = answerKey(ctx, *answers); err != nil {
		return err
	}
	if *provisional != "" {
		if exam.Provisional, err = answerKey(ctx, *provisional); err != nil {
			return err
		}
	}
	if exam.ImagePages, err = imagePages(ctx, *questions); err != nil {
		log.Warn("cannot list images, practical cases will not mention their figures", "err", err)
	}
	qs, buildProblems := examtext.Build(exam)
	problems = append(problems, buildProblems...)

	kept, err := keepReview(*out, qs)
	if err != nil {
		return err
	}
	f := content.BankFile{
		Source:    content.BankSource{Kind: content.KindINAPExam, Title: *title, Reference: *ref, URL: *url},
		Questions: qs,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep code such as <input> readable
	enc.SetIndent("", "  ")
	if err := enc.Encode(f); err != nil {
		return err
	}
	// Check the result the way the server will read it.
	if _, err := content.ParseBankFile(buf.Bytes()); err != nil {
		return err
	}
	if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
		return err
	}

	annulled := 0
	for _, q := range qs {
		if q.Annulled {
			annulled++
		}
	}
	log.Info("bank file written", "file", *out, "questions", len(qs), "annulled", annulled,
		"review_kept", kept, "problems", len(problems))
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "  problema:", p)
	}
	return nil
}

// keepReview copies topics and status from the bank file being replaced
// to the questions whose text and answer did not change, so re-importing an
// exam keeps its review. A practical case's statement before the question
// does not count: notes added to it change nothing that was checked. It
// returns how many were kept.
func keepReview(path string, qs []content.BankQuestion) (int, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	old, err := content.ParseBankFile(b)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", path, err)
	}
	byKey := map[string]content.BankQuestion{}
	for _, q := range old.Questions {
		byKey[q.Key] = q
	}
	kept := 0
	for i, q := range qs {
		o, ok := byKey[q.Key]
		if ok && questionPart(o.Stem) == questionPart(q.Stem) && o.Options == q.Options && o.Correct == q.Correct &&
			o.Annulled == q.Annulled {
			qs[i].Topics, qs[i].Status = o.Topics, o.Status
			kept++
		}
	}
	return kept, nil
}

// questionPart drops the practical-case statement that opens a stem.
func questionPart(stem string) string {
	if i := strings.LastIndex(stem, "\n\n"); i >= 0 {
		return stem[i+2:]
	}
	return stem
}

func pdfText(ctx context.Context, path string) (string, error) {
	out, err := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", path, "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext %s: %w (needs poppler-utils)", path, err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return "", fmt.Errorf("%s has no text: it is probably scanned", path)
	}
	return string(out), nil
}

func answerKey(ctx context.Context, path string) (examtext.AnswerKey, error) {
	text, err := pdfText(ctx, path)
	if err != nil {
		return nil, err
	}
	key, err := examtext.ParseAnswerKey(text)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return key, nil
}

// imagePages lists the pages with images, from the table printed by
// pdfimages -list (first column: page number).
func imagePages(ctx context.Context, path string) (map[int]bool, error) {
	out, err := exec.CommandContext(ctx, "pdfimages", "-list", path).Output()
	if err != nil {
		return nil, err
	}
	pages := map[int]bool{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) > 2 && fields[2] == "image" {
			if n, err := strconv.Atoi(fields[0]); err == nil {
				pages[n] = true
			}
		}
	}
	return pages, sc.Err()
}
