package users

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/alexzafra13/tai_tests/internal/validate"
)

// First administrator. A fresh install has no administrator that can log
// in (migrations create one without a password so existing data has an
// owner). Until one is set up, the app shows a setup screen; alternatively
// TAI_ADMIN_USER / TAI_ADMIN_PASSWORD set it up on start.

// ErrAlreadySetUp is returned when setup is attempted after an
// administrator already exists.
var ErrAlreadySetUp = errors.New("an administrator already exists")

// NeedsSetup reports whether no administrator can log in yet.
func (s *Store) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := usableAdmins(ctx, s.db)
	return n == 0, err
}

func usableAdmins(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE role = 'admin' AND active = 1 AND password_hash <> ''`).Scan(&n)
	return n, err
}

type SetupInput struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

// Setup creates the first administrator. It only works while NeedsSetup is
// true, and checks it inside the transaction so two concurrent requests
// cannot both succeed.
func (s *Store) Setup(ctx context.Context, in SetupInput) (User, error) {
	in.Username = normalizeUsername(in.Username)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	v := validate.Errors{}
	if !usernamePattern.MatchString(in.Username) {
		v["username"] = "Entre 3 y 32 caracteres: letras minúsculas, números, punto, guion o guion bajo"
	}
	validatePassword(v, "password", in.Password)
	if err := v.Err(); err != nil {
		return User{}, err
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		return User{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	if n, err := usableAdmins(ctx, tx); err != nil {
		return User{}, err
	} else if n > 0 {
		return User{}, ErrAlreadySetUp
	}

	// Reuse the administrator created by the migrations, so data that was
	// already there keeps its owner.
	now := s.timestamp()
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE role = 'admin' ORDER BY id LIMIT 1`).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		err = tx.QueryRowContext(ctx, `INSERT INTO users (username, display_name, password_hash, role, created_at, updated_at)
			VALUES (?, ?, ?, 'admin', ?, ?) RETURNING id`, in.Username, in.DisplayName, hash, now, now).Scan(&id)
	case err == nil:
		_, err = tx.ExecContext(ctx, `UPDATE users SET username = ?, display_name = ?, password_hash = ?, active = 1,
			updated_at = ? WHERE id = ?`, in.Username, in.DisplayName, hash, now, id)
	}
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return User{}, validate.Errors{"username": "Ya existe un usuario con ese nombre"}
		}
		return User{}, err
	}
	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	return s.Get(ctx, id)
}

// EnsureAdmin sets up the first administrator from the environment when a
// password is given and no administrator exists yet. It reports whether it
// created one; without a password it leaves setup to the app.
func (s *Store) EnsureAdmin(ctx context.Context, username, password string) (bool, error) {
	if password == "" {
		return false, nil
	}
	if username == "" {
		username = "admin"
	}
	_, err := s.Setup(ctx, SetupInput{Username: username, Password: password})
	if errors.Is(err, ErrAlreadySetUp) {
		return false, nil
	}
	return err == nil, err
}
