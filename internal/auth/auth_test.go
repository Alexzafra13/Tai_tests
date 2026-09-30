package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexzafra13/tai_tests/internal/db"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return NewService(d, "correct-horse", time.Hour)
}

func TestLoginAndSession(t *testing.T) {
	ctx := context.Background()
	s := newTestService(t)

	if _, _, err := s.Login(ctx, "wrong"); !errors.Is(err, ErrBadPassword) {
		t.Fatalf("wrong password: err = %v, want ErrBadPassword", err)
	}

	token, _, err := s.Login(ctx, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.Valid(ctx, token); err != nil || !ok {
		t.Fatalf("Valid(token) = %v, %v; want true", ok, err)
	}
	if ok, _ := s.Valid(ctx, "forged"); ok {
		t.Fatal("forged token accepted")
	}

	if err := s.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.Valid(ctx, token); ok {
		t.Fatal("token still valid after logout")
	}
}

func TestSessionExpires(t *testing.T) {
	ctx := context.Background()
	s := newTestService(t)
	now := time.Now()
	s.now = func() time.Time { return now }

	token, _, err := s.Login(ctx, "correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	if ok, _ := s.Valid(ctx, token); ok {
		t.Fatal("expired session accepted")
	}
}

func TestRateLimit(t *testing.T) {
	ctx := context.Background()
	s := newTestService(t)
	now := time.Now()
	s.now = func() time.Time { return now }

	for i := 0; i < 10; i++ {
		if _, _, err := s.Login(ctx, "wrong"); !errors.Is(err, ErrBadPassword) {
			t.Fatalf("attempt %d: err = %v", i, err)
		}
	}
	// Even the right password is refused while locked out.
	if _, _, err := s.Login(ctx, "correct-horse"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}

	now = now.Add(16 * time.Minute)
	if _, _, err := s.Login(ctx, "correct-horse"); err != nil {
		t.Fatalf("after window: %v", err)
	}
}
