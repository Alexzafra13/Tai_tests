// Command tai is the TAI study app: HTTP server plus content pipeline
// subcommands.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/config"
	"github.com/alexzafra13/tai_tests/internal/db"
	"github.com/alexzafra13/tai_tests/internal/server"
	"github.com/alexzafra13/tai_tests/web"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `Usage: tai <command>

Commands:
  serve     Run migrations and start the web server
  migrate   Apply pending database migrations
  version   Print the version

Configuration is read from environment variables; see .env.example.
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
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	d, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	applied, err := db.Migrate(ctx, d)
	for _, name := range applied {
		log.Info("applied migration", "name", name)
	}
	return err
}

func runServe(ctx context.Context, log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.RequireServe(); err != nil {
		return err
	}

	d, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	applied, err := db.Migrate(ctx, d)
	if err != nil {
		return err
	}
	for _, name := range applied {
		log.Info("applied migration", "name", name)
	}

	srv := server.New(auth.NewService(d, cfg.Password, cfg.SessionTTL), cfg.CookieSecure, web.Dist(), log)
	log.Info("starting tai", "version", version, "db", cfg.DBPath)
	return server.Run(ctx, cfg.Addr, srv.Handler(), log)
}
