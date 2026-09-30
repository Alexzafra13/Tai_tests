// Package auth implements single-user password login with server-side
// sessions stored in SQLite.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const CookieName = "tai_session"

var (
	ErrBadPassword = errors.New("incorrect password")
	ErrRateLimited = errors.New("too many failed attempts")
)

const timeFormat = "2006-01-02T15:04:05.000Z"

type Service struct {
	db       *sql.DB
	password [32]byte
	ttl      time.Duration
	limiter  *limiter
	now      func() time.Time
}

func NewService(db *sql.DB, password string, ttl time.Duration) *Service {
	return &Service{
		db:       db,
		password: sha256.Sum256([]byte(password)),
		ttl:      ttl,
		limiter:  newLimiter(10, 15*time.Minute),
		now:      time.Now,
	}
}

// Login checks the password and, if correct, creates a session and returns
// its token. Failed attempts are rate limited globally: there is only one
// account, so per-client limits would add nothing but complexity.
func (s *Service) Login(ctx context.Context, password string) (token string, expires time.Time, err error) {
	now := s.now()
	if !s.limiter.allow(now) {
		return "", time.Time{}, ErrRateLimited
	}
	// Comparing fixed-size hashes keeps the comparison constant-time
	// regardless of input length.
	got := sha256.Sum256([]byte(password))
	if subtle.ConstantTimeCompare(got[:], s.password[:]) != 1 {
		s.limiter.fail(now)
		return "", time.Time{}, ErrBadPassword
	}
	s.limiter.reset()

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expires = now.Add(s.ttl).UTC()

	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, now.UTC().Format(timeFormat)); err != nil {
		return "", time.Time{}, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, expires_at) VALUES (?, ?)`,
		hashToken(token), expires.Format(timeFormat)); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// Valid reports whether token belongs to an unexpired session.
func (s *Service) Valid(ctx context.Context, token string) (bool, error) {
	if token == "" {
		return false, nil
	}
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sessions WHERE token_hash = ? AND expires_at > ?`,
		hashToken(token), s.now().UTC().Format(timeFormat)).Scan(&n)
	return n == 1, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(token))
	return err
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// limiter allows at most max failures within window.
type limiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	failures []time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{max: max, window: window}
}

func (l *limiter) prune(now time.Time) {
	i := 0
	for i < len(l.failures) && now.Sub(l.failures[i]) >= l.window {
		i++
	}
	l.failures = l.failures[i:]
}

func (l *limiter) allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(now)
	return len(l.failures) < l.max
}

func (l *limiter) fail(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures = append(l.failures, now)
}

func (l *limiter) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures = nil
}
