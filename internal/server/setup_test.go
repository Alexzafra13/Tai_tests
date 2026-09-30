package server

import (
	"net/http"
	"testing"
)

func TestFirstRunSetup(t *testing.T) {
	env, _ := newBareEnv(t)
	base := env.ts.URL
	c := newClient()

	var status struct{ Needed bool }
	if doJSON(t, c, "GET", base+"/api/setup", nil, &status); !status.Needed {
		t.Fatal("fresh install should need setup")
	}

	var verr struct{ Fields map[string]string }
	if code := doJSON(t, c, "POST", base+"/api/setup", map[string]any{"username": "a", "password": "x"}, &verr); code != http.StatusUnprocessableEntity || verr.Fields["password"] == "" {
		t.Fatalf("invalid setup: %d %+v", code, verr)
	}

	var me struct{ Username, Role string }
	if code := doJSON(t, c, "POST", base+"/api/setup", map[string]any{"username": "Alex", "display_name": "Alex", "password": "alex-password"}, &me); code != http.StatusOK || me.Role != "admin" || me.Username != "alex" {
		t.Fatalf("setup: %d %+v", code, me)
	}
	// The setup logs the administrator in.
	if code := doJSON(t, c, "GET", base+"/api/users", nil, nil); code != http.StatusOK {
		t.Errorf("admin endpoint after setup: %d", code)
	}

	// From now on setup is closed, even for someone else on the network.
	other := newClient()
	if doJSON(t, other, "GET", base+"/api/setup", nil, &status); status.Needed {
		t.Error("setup still offered")
	}
	if code := doJSON(t, other, "POST", base+"/api/setup", map[string]any{"username": "intruso", "password": "intruso-password"}, nil); code != http.StatusConflict {
		t.Errorf("second setup: %d, want 409", code)
	}
}
