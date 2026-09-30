package users

import (
	"context"
	"errors"
	"testing"

	"github.com/alexzafra13/tai_tests/internal/db"
	"github.com/alexzafra13/tai_tests/internal/validate"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return NewStore(d)
}

func TestEnsureAdminOnFirstStart(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	if _, err := s.EnsureAdmin(ctx, "admin", ""); err == nil {
		t.Fatal("first start without password should fail")
	}
	created, err := s.EnsureAdmin(ctx, "Alex", "first-password")
	if err != nil || !created {
		t.Fatalf("EnsureAdmin = %v, %v", created, err)
	}
	u, err := s.Authenticate(ctx, "alex", "first-password")
	if err != nil || u.ID != 1 || !u.IsAdmin() {
		t.Fatalf("admin login: %+v, %v", u, err)
	}

	// Later starts leave the account alone, whatever the environment says.
	if created, err := s.EnsureAdmin(ctx, "other", "another-password"); err != nil || created {
		t.Fatalf("second EnsureAdmin = %v, %v", created, err)
	}
	if _, err := s.Authenticate(ctx, "alex", "another-password"); !errors.Is(err, ErrBadCredentials) {
		t.Error("environment password overwrote the admin's password")
	}
}

func TestCreateAndAuthenticate(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	s.EnsureAdmin(ctx, "admin", "admin-password")

	_, err := s.Create(ctx, CreateInput{Username: "A", Password: "short", Role: "root"})
	var v validate.Errors
	if !errors.As(err, &v) || v["username"] == "" || v["password"] == "" || v["role"] == "" {
		t.Fatalf("invalid create: %v", err)
	}

	id, err := s.Create(ctx, CreateInput{Username: " Maria.G ", DisplayName: "María", Password: "maria-password", Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, CreateInput{Username: "maria.g", Password: "whatever-123", Role: RoleUser}); err == nil {
		t.Error("duplicate username accepted")
	}
	u, err := s.Authenticate(ctx, "MARIA.G", "maria-password")
	if err != nil || u.ID != id || u.IsAdmin() || u.DisplayName != "María" {
		t.Fatalf("user login: %+v, %v", u, err)
	}
	if _, err := s.Authenticate(ctx, "maria.g", "wrong-password"); !errors.Is(err, ErrBadCredentials) {
		t.Error("wrong password accepted")
	}
	if _, err := s.Authenticate(ctx, "nobody", "whatever-123"); !errors.Is(err, ErrBadCredentials) {
		t.Error("unknown user accepted")
	}

	if err := s.Update(ctx, id, UpdateInput{DisplayName: "María", Role: RoleUser, Active: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, "maria.g", "maria-password"); !errors.Is(err, ErrBadCredentials) {
		t.Error("inactive user can log in")
	}

	if err := s.SetPassword(ctx, id, "new-password"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.CheckPassword(ctx, id, "new-password"); !ok {
		t.Error("password not changed")
	}
}

func TestCannotRemoveLastAdmin(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	s.EnsureAdmin(ctx, "admin", "admin-password")

	if err := s.Update(ctx, 1, UpdateInput{Role: RoleUser, Active: true}); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("demoting last admin: err = %v", err)
	}
	if err := s.Update(ctx, 1, UpdateInput{Role: RoleAdmin, Active: false}); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("deactivating last admin: err = %v", err)
	}

	id, _ := s.Create(ctx, CreateInput{Username: "second", Password: "second-password", Role: RoleAdmin})
	if err := s.Update(ctx, 1, UpdateInput{Role: RoleUser, Active: true}); err != nil {
		t.Fatalf("demoting with another admin: %v", err)
	}
	if u, _ := s.Get(ctx, id); !u.IsAdmin() {
		t.Error("second admin lost role")
	}
}
