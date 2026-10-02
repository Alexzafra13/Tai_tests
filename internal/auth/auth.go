// Package auth handles login sessions: it checks credentials, stores
// session tokens (hashed) in SQLite and exposes the current user to handlers.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/alexzafra13/tai_tests/internal/db"

	"github.com/alexzafra13/tai_tests/internal/users"
)

const CookieName = "tai_session"

var (
	ErrBadCredentials = errors.New("incorrect username or password")
	ErrRateLimited    = errors.New("too many failed attempts")
	ErrWrongPassword  = errors.New("current password is wrong")
)

type Service struct {
	db      *sql.DB
	users   *users.Store
	ttl     time.Duration
	limiter *limiter
	now     func() time.Time
}

func NewService(db *sql.DB, us *users.Store, ttl time.Duration) *Service {
	return &Service{db: db, users: us, ttl: ttl, limiter: newLimiter(10, 15*time.Minute), now: time.Now}
}

type Session struct {
	Token   string
	Expires time.Time
	User    users.User
}

// Login checks the credentials and opens a session. Failed attempts are
// limited per username, so guessing one account's password cannot lock the
// others out.
func (s *Service) Login(ctx context.Context, username, password string) (Session, error) {
	now := s.now()
	key := users.NormalizeUsername(username)
	if !s.limiter.allow(key, now) {
		return Session{}, ErrRateLimited
	}
	u, err := s.users.Authenticate(ctx, username, password)
	if errors.Is(err, users.ErrBadCredentials) {
		s.limiter.fail(key, now)
		return Session{}, ErrBadCredentials
	} else if err != nil {
		return Session{}, err
	}
	s.limiter.reset(key)
	return s.OpenSession(ctx, u)
}

// OpenSession starts a session for an already authenticated user, e.g.
// right after the first-run setup.
func (s *Service) OpenSession(ctx context.Context, u users.User) (Session, error) {
	now := s.now()
	token, err := newToken()
	if err != nil {
		return Session{}, err
	}
	expires := now.Add(s.ttl).UTC()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, db.Timestamp(now)); err != nil {
		return Session{}, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		hashToken(token), u.ID, db.Timestamp(expires)); err != nil {
		return Session{}, err
	}
	return Session{Token: token, Expires: expires, User: u}, nil
}

// UserForToken returns the user of an unexpired session. Deactivated
// accounts lose access immediately, even with an open session.
func (s *Service) UserForToken(ctx context.Context, token string) (users.User, bool, error) {
	if token == "" {
		return users.User{}, false, nil
	}
	var userID int64
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM sessions WHERE token_hash = ? AND expires_at > ?`,
		hashToken(token), db.Timestamp(s.now())).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return users.User{}, false, nil
	} else if err != nil {
		return users.User{}, false, err
	}
	u, err := s.users.Get(ctx, userID)
	if errors.Is(err, users.ErrNotFound) || (err == nil && !u.Active) {
		return users.User{}, false, nil
	}
	return u, err == nil, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}

// ChangePassword changes the caller's own password after checking the
// current one, and closes their other sessions.
func (s *Service) ChangePassword(ctx context.Context, userID int64, currentToken, current, next string) error {
	ok, err := s.users.CheckPassword(ctx, userID, current)
	if err != nil {
		return err
	}
	if !ok {
		return ErrWrongPassword
	}
	if err := s.users.SetPassword(ctx, userID, next); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ? AND token_hash <> ?`, userID, hashToken(currentToken))
	return err
}

// EndSessions closes all sessions of a user, e.g. after an administrator
// resets their password.
func (s *Service) EndSessions(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func newToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type ctxKey struct{}

func WithUser(ctx context.Context, u users.User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// CurrentUser returns the authenticated user of a request. Handlers behind
// the auth middleware can rely on it being set.
func CurrentUser(ctx context.Context) users.User {
	u, _ := ctx.Value(ctxKey{}).(users.User)
	return u
}
