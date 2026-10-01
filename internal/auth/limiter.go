package auth

import (
	"sync"
	"time"
)

// maxTrackedKeys bounds memory when someone tries many made-up usernames:
// past it, keys with no recent failures are swept.
const maxTrackedKeys = 10000

// limiter allows at most max failures per key within window.
type limiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	failures map[string][]time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{max: max, window: window, failures: map[string][]time.Time{}}
}

// prune drops failures older than the window, and forgets keys with none
// left so the map cannot grow without bound.
func (l *limiter) prune(key string, now time.Time) []time.Time {
	f := l.failures[key]
	i := 0
	for i < len(f) && now.Sub(f[i]) >= l.window {
		i++
	}
	f = f[i:]
	if len(f) == 0 {
		delete(l.failures, key)
	} else {
		l.failures[key] = f
	}
	return f
}

func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(key, now)) < l.max
}

func (l *limiter) fail(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.failures) >= maxTrackedKeys {
		for k := range l.failures {
			l.prune(k, now)
		}
	}
	l.failures[key] = append(l.prune(key, now), now)
}

func (l *limiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}
