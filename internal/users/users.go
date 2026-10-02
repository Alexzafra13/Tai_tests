// Package users manages accounts: administrators, who manage content and
// other accounts, and users, who study. There is no self sign-up; accounts
// are created by an administrator.
package users

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexzafra13/tai_tests/internal/db"

	"golang.org/x/crypto/bcrypt"

	"github.com/alexzafra13/tai_tests/internal/validate"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) valid() bool { return r == RoleAdmin || r == RoleUser }

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        Role   `json:"role"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
}

func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

var (
	ErrNotFound       = errors.New("user not found")
	ErrBadCredentials = errors.New("bad credentials")
	// ErrLastAdmin protects against locking everyone out of administration.
	ErrLastAdmin = errors.New("at least one active administrator is required")
)

const MinPasswordLength = 8

var usernamePattern = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db, now: time.Now}
}

func (s *Store) timestamp() string { return db.Timestamp(s.now()) }

const userColumns = `id, username, display_name, role, active, created_at`

// scanUser reads userColumns into u, then any extra columns into extra.
func scanUser(row interface{ Scan(...any) error }, u *User, extra ...any) error {
	return row.Scan(append([]any{&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Active, &u.CreatedAt}, extra...)...)
}

func (s *Store) Get(ctx context.Context, id int64) (User, error) {
	var u User
	err := scanUser(s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id), &u)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) GetByUsername(ctx context.Context, username string) (User, error) {
	var u User
	err := scanUser(s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE username = ?`,
		NormalizeUsername(username)), &u)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) List(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+userColumns+` FROM users ORDER BY role, username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := scanUser(rows, &u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

type CreateInput struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        Role   `json:"role"`
}

// NormalizeUsername is the stored form of a username: usernames are
// case-insensitive and ignore surrounding spaces.
func NormalizeUsername(u string) string { return strings.ToLower(strings.TrimSpace(u)) }

func validateUsername(v validate.Errors, username string) {
	if !usernamePattern.MatchString(username) {
		v["username"] = "Entre 3 y 32 caracteres: letras minúsculas, números, punto, guion o guion bajo"
	}
}

func usernameTaken() error { return validate.Errors{"username": "Ya existe un usuario con ese nombre"} }

func validatePassword(v validate.Errors, field, password string) {
	if utf8.RuneCountInString(password) < MinPasswordLength {
		v[field] = "La contraseña debe tener al menos 8 caracteres"
	}
}

func (s *Store) Create(ctx context.Context, in CreateInput) (int64, error) {
	in.Username = NormalizeUsername(in.Username)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	v := validate.Errors{}
	validateUsername(v, in.Username)
	if !in.Role.valid() {
		v["role"] = "Rol no válido"
	}
	validatePassword(v, "password", in.Password)
	if err := v.Err(); err != nil {
		return 0, err
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		return 0, err
	}
	now := s.timestamp()
	var id int64
	err = s.db.QueryRowContext(ctx, `INSERT INTO users (username, display_name, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id`, in.Username, in.DisplayName, hash, in.Role, now, now).Scan(&id)
	if db.IsUnique(err) {
		return 0, usernameTaken()
	}
	return id, err
}

type UpdateInput struct {
	DisplayName string `json:"display_name"`
	Role        Role   `json:"role"`
	Active      bool   `json:"active"`
}

// Update changes an account's name, role and active flag. It refuses to
// leave the app without an active administrator.
func (s *Store) Update(ctx context.Context, id int64, in UpdateInput) error {
	if !in.Role.valid() {
		return validate.Errors{"role": "Rol no válido"}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `UPDATE users SET display_name = ?, role = ?, active = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(in.DisplayName), in.Role, in.Active, s.timestamp(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := requireActiveAdmin(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func requireActiveAdmin(ctx context.Context, tx *sql.Tx) error {
	n, err := usableAdmins(ctx, tx)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrLastAdmin
	}
	return nil
}

func (s *Store) SetPassword(ctx context.Context, id int64, password string) error {
	v := validate.Errors{}
	validatePassword(v, "password", password)
	if err := v.Err(); err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, hash, s.timestamp(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// dummyHash is compared against when the username does not exist, so a
// login attempt takes the same time whether or not the account exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

// Authenticate returns the active user matching the credentials. Accounts
// without a password (the admin row created by migrations) never match.
func (s *Store) Authenticate(ctx context.Context, username, password string) (User, error) {
	var u User
	var hash string
	err := scanUser(s.db.QueryRowContext(ctx, `SELECT `+userColumns+`, password_hash FROM users WHERE username = ?`,
		NormalizeUsername(username)), &u, &hash)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && hash == "") {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return User{}, ErrBadCredentials
	} else if err != nil {
		return User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || !u.Active {
		return User{}, ErrBadCredentials
	}
	return u, nil
}

func (s *Store) CheckPassword(ctx context.Context, id int64, password string) (bool, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = ?`, id).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	} else if err != nil {
		return false, err
	}
	return hash != "" && bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil, nil
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
