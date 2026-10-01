package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexzafra13/tai_tests/internal/db"
	"github.com/alexzafra13/tai_tests/internal/users"
)

type fixture struct {
	auth  *Service
	users *users.Store
	now   time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	us := users.NewStore(d)
	if _, err := us.EnsureAdmin(ctx, "admin", "admin-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := us.Create(ctx, users.CreateInput{Username: "ana", Password: "ana-password", Role: users.RoleUser}); err != nil {
		t.Fatal(err)
	}
	f := &fixture{auth: NewService(d, us, time.Hour), users: us, now: time.Now()}
	f.auth.now = func() time.Time { return f.now }
	return f
}

func TestLoginAndSession(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	if _, err := f.auth.Login(ctx, "ana", "wrong"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("wrong password: err = %v", err)
	}
	sess, err := f.auth.Login(ctx, "ana", "ana-password")
	if err != nil {
		t.Fatal(err)
	}
	u, ok, err := f.auth.UserForToken(ctx, sess.Token)
	if err != nil || !ok || u.Username != "ana" || u.IsAdmin() {
		t.Fatalf("UserForToken = %+v, %v, %v", u, ok, err)
	}
	if _, ok, _ := f.auth.UserForToken(ctx, "forged"); ok {
		t.Fatal("forged token accepted")
	}

	if err := f.auth.Logout(ctx, sess.Token); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := f.auth.UserForToken(ctx, sess.Token); ok {
		t.Fatal("token valid after logout")
	}
}

func TestSessionExpiresAndDeactivation(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	sess, _ := f.auth.Login(ctx, "ana", "ana-password")

	if err := f.users.Update(ctx, sess.User.ID, users.UpdateInput{Role: users.RoleUser, Active: false}); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := f.auth.UserForToken(ctx, sess.Token); ok {
		t.Fatal("deactivated user keeps access")
	}

	admin, _ := f.auth.Login(ctx, "admin", "admin-password")
	f.now = f.now.Add(2 * time.Hour)
	if _, ok, _ := f.auth.UserForToken(ctx, admin.Token); ok {
		t.Fatal("expired session accepted")
	}
}

func TestRateLimitIsPerUsername(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	for i := 0; i < 10; i++ {
		if _, err := f.auth.Login(ctx, "ana", "wrong"); !errors.Is(err, ErrBadCredentials) {
			t.Fatalf("attempt %d: err = %v", i, err)
		}
	}
	if _, err := f.auth.Login(ctx, "ana", "ana-password"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("locked account: err = %v, want ErrRateLimited", err)
	}
	// Someone guessing Ana's password does not lock the admin out.
	if _, err := f.auth.Login(ctx, "admin", "admin-password"); err != nil {
		t.Fatalf("other account locked too: %v", err)
	}

	f.now = f.now.Add(16 * time.Minute)
	if _, err := f.auth.Login(ctx, "ANA", "ana-password"); err != nil {
		t.Fatalf("after window: %v", err)
	}
}

func TestChangePasswordClosesOtherSessions(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	phone, _ := f.auth.Login(ctx, "ana", "ana-password")
	laptop, _ := f.auth.Login(ctx, "ana", "ana-password")

	if err := f.auth.ChangePassword(ctx, phone.User.ID, phone.Token, "wrong", "brand-new-password"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("wrong current password: err = %v", err)
	}
	if err := f.auth.ChangePassword(ctx, phone.User.ID, phone.Token, "ana-password", "brand-new-password"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := f.auth.UserForToken(ctx, phone.Token); !ok {
		t.Error("current session closed")
	}
	if _, ok, _ := f.auth.UserForToken(ctx, laptop.Token); ok {
		t.Error("other session still open")
	}
	if _, err := f.auth.Login(ctx, "ana", "brand-new-password"); err != nil {
		t.Errorf("login with new password: %v", err)
	}
}
