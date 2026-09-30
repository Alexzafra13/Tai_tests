package auth

import (
	"fmt"
	"testing"
	"time"
)

func TestLimiterForgetsOldKeys(t *testing.T) {
	l := newLimiter(3, time.Minute)
	start := time.Now()
	for i := range maxTrackedKeys {
		l.fail(fmt.Sprintf("user%d", i), start)
	}
	// Once the old failures are out of the window, a new failure sweeps them.
	l.fail("fresh", start.Add(2*time.Minute))
	if n := len(l.failures); n != 1 {
		t.Fatalf("tracked keys = %d, want 1", n)
	}
}
