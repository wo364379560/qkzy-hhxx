package ratelimit

import (
	"sync"
	"time"
)

const maxSecurityLogSampleKeys = 128

type logSampleEntry struct {
	last       time.Time
	suppressed int64
}

// keyedLogSampler keeps independent sampling windows for a bounded set of
// finite security-event dimensions. Request-controlled subjects and IPs must
// never be included in its keys.
type keyedLogSampler struct {
	mu      sync.Mutex
	entries map[string]logSampleEntry
	window  time.Duration
	maxKeys int
}

func newKeyedLogSampler(window time.Duration, maxKeys int) *keyedLogSampler {
	if maxKeys <= 0 {
		maxKeys = 1
	}
	return &keyedLogSampler{
		entries: make(map[string]logSampleEntry),
		window:  window,
		maxKeys: maxKeys,
	}
}

// sample returns emit=true for the first event in a key's window. The
// suppressed count belongs only to that key and is reported on its next emit.
func (s *keyedLogSampler) sample(key string, now time.Time) (emit bool, suppressed int64) {
	if s == nil || s.window <= 0 {
		return true, 0
	}
	key = normalizeSubject(key)
	if key == "" {
		key = "unknown"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[key]
	if exists && now.Sub(entry.last) < s.window {
		entry.suppressed++
		s.entries[key] = entry
		return false, 0
	}

	if !exists && len(s.entries) >= s.maxKeys {
		s.evictOldest()
	}

	s.entries[key] = logSampleEntry{last: now}
	return true, entry.suppressed
}

func (s *keyedLogSampler) evictOldest() {
	var oldestKey string
	var oldest time.Time
	for key, entry := range s.entries {
		if oldestKey == "" || entry.last.Before(oldest) {
			oldestKey = key
			oldest = entry.last
		}
	}
	if oldestKey != "" {
		delete(s.entries, oldestKey)
	}
}

func (s *keyedLogSampler) len() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func limitedLogSampleKey(action, scope string) string {
	return "limited|" + normalizeSubject(action) + "|" + normalizeSubject(scope)
}

func overloadLogSampleKey(action, reason string) string {
	return "overload|" + normalizeSubject(action) + "|" + normalizeSubject(reason)
}

func redisWarningSampleKey(scope string) string {
	return "redis|" + normalizeSubject(scope)
}
