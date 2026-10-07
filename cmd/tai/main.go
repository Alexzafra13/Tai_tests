// Command tai is the TAI study app: HTTP server plus content pipeline
// subcommands.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	// Time zone data inside the binary: statistics use each browser's zone.
	_ "time/tzdata"

	"github.com/alexzafra13/tai_tests/data"
	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/config"
	"github.com/alexzafra13/tai_tests/internal/content"
	"github.com/alexzafra13/tai_tests/internal/db"
	"github.com/alexzafra13/tai_tests/internal/quiz"
	"github.com/alexzafra13/tai_tests/internal/server"
	"github.com/alexzafra13/tai_tests/internal/settings"
	"github.com/alexzafra13/tai_tests/internal/srs"
	"github.com/alexzafra13/tai_tests/internal/stats"
	"github.com/alexzafra13/tai_tests/internal/users"
	"github.com/alexzafra13/tai_tests/web"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `Usage: tai <command>

Commands:
  serve          Run migrations and start the web server
  migrate        Apply pending database migrations
  load-syllabus  Load or update the syllabus from a JSON file
  add-source     Add a source document (law, technical doc, exam)
  import-exam    Write a question bank file from an INAP exam (PDFs)
  fetch-law      Download a law's consolidated text from the BOE (data/laws)
  refresh-laws   Download again every law of data/laws
  user           Manage accounts: user list | user add | user passwd
  version        Print the version

Run "tai <command> -h" for the flags of a command.

Nothing needs configuring. Optional environment variables: TAI_ADDR,
TAI_DB_PATH, TAI_SESSION_TTL, TAI_ADMIN_USER and TAI_ADMIN_PASSWORD.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	switch os.Args[1] {
	case "serve":
		err = runServe(ctx, log)
	case "migrate":
		err = runMigrate(ctx, log)
	case "load-syllabus":
		err = runLoadSyllabus(ctx, log, os.Args[2:])
	case "add-source":
		err = runAddSource(ctx, log, os.Args[2:])
	case "import-exam":
		err = runImportExam(ctx, log, os.Args[2:])
	case "fetch-law":
		err = runFetchLaw(ctx, log, os.Args[2:])
	case "refresh-laws":
		err = runRefreshLaws(ctx, log, os.Args[2:])
	case "user":
		err = runUser(ctx, log, os.Args[2:])
	case "version":
		fmt.Println(version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		log.Error(os.Args[1]+" failed", "err", err)
		os.Exit(1)
	}
}

func runMigrate(ctx context.Context, log *slog.Logger) error {
	d, err := openDB(ctx, log)
	if err != nil {
		return err
	}
	return d.Close()
}

// openDB loads the config, opens the database and applies pending
// migrations, so every subcommand works on an up-to-date schema.
func openDB(ctx context.Context, log *slog.Logger) (*sql.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	d, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	applied, err := db.Migrate(ctx, d)
	for _, name := range applied {
		log.Info("applied migration", "name", name)
	}
	if err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func runServe(ctx context.Context, log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	d, err := openDB(ctx, log)
	if err != nil {
		return err
	}
	defer d.Close()

	us := users.NewStore(d)
	created, err := us.EnsureAdmin(ctx, cfg.AdminUser, cfg.AdminPassword)
	if err != nil {
		return err
	}
	if created {
		log.Info("administrator created from TAI_ADMIN_USER / TAI_ADMIN_PASSWORD", "username", cfg.AdminUser)
	}
	if need, err := us.NeedsSetup(ctx); err != nil {
		return err
	} else if need {
		log.Info("first start: open the app in a browser to create the administrator account", "addr", cfg.Addr)
	}

	cs := content.NewStore(d)
	if err := loadBank(ctx, log, cs); err != nil {
		return err
	}

	sr := srs.NewStore(d)
	srv := server.New(server.Deps{
		Auth:    auth.NewService(d, us, cfg.SessionTTL),
		Users:   us,
		Content: cs,
		Quiz:    quiz.NewStore(d, settings.NewStore(d), sr),
		SRS:     sr,
		Stats:   stats.NewStore(d),
		Static:  web.Dist(),
		Log:     log,
	})
	log.Info("starting tai", "version", version, "db", cfg.DBPath)
	return server.Run(ctx, cfg.Addr, srv.Handler(), log)
}

// loadBank adds the bundled official content this installation does not
// have yet. A new installation first gets the bundled syllabus, so the
// laws can be linked to their topics and checked questions can arrive
// published with theirs.
func loadBank(ctx context.Context, log *slog.Logger, cs *content.Store) error {
	blocks, err := cs.Syllabus(ctx)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		f, err := content.ParseSyllabus(data.Syllabus())
		if err != nil {
			return err
		}
		res, err := cs.LoadSyllabus(ctx, f)
		if err != nil {
			return err
		}
		log.Info("syllabus loaded", "name", f.Name, "blocks", res.Blocks, "topics", res.Topics)
	}

	laws, lawTopics, err := content.ReadLaws(data.Laws())
	if err != nil {
		return err
	}
	lres, err := cs.LoadLaws(ctx, laws, lawTopics)
	if err != nil {
		return err
	}
	for _, p := range lres.Problems {
		log.Warn("law text not updated", "reason", p)
	}
	if lres.Added+lres.Updated > 0 {
		log.Info("laws loaded", "added", lres.Added, "updated", lres.Updated)
	}

	files, err := content.ReadBank(data.Bank())
	if err != nil {
		return err
	}
	res, err := cs.LoadBank(ctx, files)
	if err != nil {
		return err
	}
	for _, p := range res.Problems {
		log.Warn("bank question not loaded", "source", p.Source, "key", p.Key, "reason", p.Reason)
	}
	if res.Added > 0 {
		log.Info("question bank loaded", "added", res.Added, "published", res.Published,
			"drafts_for_review", res.Added-res.Published)
	}
	return nil
}
