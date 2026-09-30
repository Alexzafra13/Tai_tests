package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alexzafra13/tai_tests/internal/auth"
	"github.com/alexzafra13/tai_tests/internal/users"
	"github.com/alexzafra13/tai_tests/internal/validate"
)

const userUsage = `Usage:
  tai user list
  tai user add -username NAME [-name "Display name"] [-role user|admin]
  tai user passwd -username NAME

Passwords are read from standard input (one line), so they never appear in
the shell history or the process list:
  docker compose exec -T tai tai user passwd -username admin <<< 'new-password'
`

// runUser manages accounts from the command line, mainly to recover access
// (e.g. a forgotten administrator password). Day-to-day, accounts are
// managed from the app.
func runUser(ctx context.Context, log *slog.Logger, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, userUsage)
		os.Exit(2)
	}
	d, err := openDB(ctx, log)
	if err != nil {
		return err
	}
	defer d.Close()
	us := users.NewStore(d)

	switch args[0] {
	case "list":
		list, err := us.List(ctx)
		if err != nil {
			return err
		}
		for _, u := range list {
			state := "active"
			if !u.Active {
				state = "inactive"
			}
			fmt.Printf("%-4d %-20s %-6s %-8s %s\n", u.ID, u.Username, u.Role, state, u.DisplayName)
		}
		return nil

	case "add":
		fs := flag.NewFlagSet("user add", flag.ExitOnError)
		username := fs.String("username", "", "login name")
		name := fs.String("name", "", "display name")
		role := fs.String("role", "user", "user or admin")
		fs.Parse(args[1:])
		password, err := readPassword()
		if err != nil {
			return err
		}
		id, err := us.Create(ctx, users.CreateInput{Username: *username, DisplayName: *name, Password: password, Role: users.Role(*role)})
		if err != nil {
			return describe(err)
		}
		log.Info("user created", "id", id, "username", *username, "role", *role)
		return nil

	case "passwd":
		fs := flag.NewFlagSet("user passwd", flag.ExitOnError)
		username := fs.String("username", "", "login name")
		fs.Parse(args[1:])
		password, err := readPassword()
		if err != nil {
			return err
		}
		u, err := us.GetByUsername(ctx, *username)
		if err != nil {
			return fmt.Errorf("user %q: %w", *username, err)
		}
		if err := us.SetPassword(ctx, u.ID, password); err != nil {
			return describe(err)
		}
		// Close open sessions: the old password may be compromised.
		if err := auth.NewService(d, us, 0).EndSessions(ctx, u.ID); err != nil {
			return err
		}
		log.Info("password changed", "username", u.Username)
		return nil

	default:
		fmt.Fprint(os.Stderr, userUsage)
		os.Exit(2)
		return nil
	}
}

func readPassword() (string, error) {
	fmt.Fprintln(os.Stderr, "Password (one line on stdin):")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", errors.New("no password on stdin")
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// describe prints validation messages before returning the error.
func describe(err error) error {
	var v validate.Errors
	if errors.As(err, &v) {
		for field, msg := range v {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", field, msg)
		}
	}
	return err
}
