package ratelimit

import (
	"sync"
	"time"
)

const defaultMaxLocalSubjects = 100000
const localCleanupInterval = 10 * time.Second

const (
	defaultMaxProcessScopes           = 128
	defaultMaxProcessSubjectsPerScope = 2048
)

type localCounter struct {
	count     int64
	expiresAt time.Time
}

// LocalOptions configures a bounded, in-process fixed-window limiter.
// The limiter is intentionally simple: it is the last line of defence when
// Redis is disabled or temporarily unavailable, not a replacement for the
// distributed limiter.
type LocalOptions struct {
	Limit       int64
	Window      time.Duration
	MaxSubjects int
	Now         func() time.Time
}

// LocalLimiter is safe for concurrent use and keeps a bounded number of keys.
// When its key budget is exhausted, new subjects are rejected instead of
// allocating unbounded memory under a distributed random-key attack.
type LocalLimiter struct {
	mu          sync.Mutex
	entries     map[string]localCounter
	limit       int64
	window      time.Duration
	maxSubjects int
	now         func() time.Time
	nextCleanup time.Time
}

func NewLocalLimiter(options LocalOptions) *LocalLimiter {
	maxSubjects := options.MaxSubjects
	if maxSubjects <= 0 {
		maxSubjects = defaultMaxLocalSubjects
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &LocalLimiter{
		entries:     make(map[string]localCounter),
		limit:       options.Limit,
		window:      options.Window,
		maxSubjects: maxSubjects,
		now:         now,
	}
}

func (l *LocalLimiter) Check(subject string, cost int64) Decision {
	if l == nil || l.limit <= 0 || l.window <= 0 {
		return Decision{Allowed: true}
	}
	if cost <= 0 {
		cost = 1
	}
	subject = normalizeSubject(subject)
	if subject == "" {
		return Decision{Allowed: true}
	}

	rule := Rule{Scope: "local", Limit: l.limit, Window: l.window}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.entries[subject]
	if exists && !now.Before(entry.expiresAt) {
		delete(l.entries, subject)
		exists = false
	}
	if !exists && len(l.entries) >= l.maxSubjects {
		if !now.Before(l.nextCleanup) {
			l.removeExpired(now)
			l.nextCleanup = now.Add(localCleanupInterval)
		}
		if len(l.entries) >= l.maxSubjects {
			return Decision{Allowed: false, Rule: rule, Count: l.limit + cost, Limit: l.limit}
		}
	}
	if !exists {
		entry = localCounter{expiresAt: now.Add(l.window)}
	}
	entry.count += cost
	l.entries[subject] = entry
	return Decision{
		Allowed: entry.count <= l.limit,
		Rule:    rule,
		Count:   entry.count,
		Limit:   l.limit,
	}
}

func (l *LocalLimiter) Len() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

func (l *LocalLimiter) removeExpired(now time.Time) {
	for subject, entry := range l.entries {
		if !now.Before(entry.expiresAt) {
			delete(l.entries, subject)
		}
	}
}

type processCounter struct {
	count     int64
	expiresAt time.Time
}

type localScopeBucket struct {
	entries    map[string]*processCounter
	nextExpiry time.Time
}

type localRuleStore struct {
	mu                  sync.Mutex
	buckets             map[string]*localScopeBucket
	maxScopes           int
	maxSubjectsPerScope int
}

var processLimiter = localRuleStore{
	buckets:             make(map[string]*localScopeBucket),
	maxScopes:           defaultMaxProcessScopes,
	maxSubjectsPerScope: defaultMaxProcessSubjectsPerScope,
}

func (l *localRuleStore) check(subject string, cost int64, rule Rule, now time.Time) Decision {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.buckets == nil {
		l.buckets = make(map[string]*localScopeBucket)
	}
	maxScopes := l.maxScopes
	if maxScopes <= 0 {
		maxScopes = defaultMaxProcessScopes
	}
	maxSubjects := l.maxSubjectsPerScope
	if maxSubjects <= 0 {
		maxSubjects = defaultMaxProcessSubjectsPerScope
	}

	scope := sanitize(rule.Scope)
	bucket := l.buckets[scope]
	if bucket == nil {
		// Scopes are compiled server rules rather than client input. If a future
		// caller creates unbounded dynamic scopes, do not let it allocate memory.
		if len(l.buckets) >= maxScopes {
			return Decision{Allowed: false, Rule: rule, Count: rule.Limit + cost, Limit: rule.Limit}
		}
		bucket = &localScopeBucket{entries: make(map[string]*processCounter)}
		l.buckets[scope] = bucket
	}

	entry, exists := bucket.entries[subject]
	if exists && (entry == nil || !now.Before(entry.expiresAt)) {
		delete(bucket.entries, subject)
		entry = nil
		exists = false
	}
	if !exists && len(bucket.entries) >= maxSubjects {
		// Avoid an O(maxSubjects) scan for every random key once the bucket is
		// full. nextExpiry tells us whether any entry can possibly be reclaimed.
		if !bucket.nextExpiry.IsZero() && !now.Before(bucket.nextExpiry) {
			bucket.removeExpired(now)
		}
		if len(bucket.entries) >= maxSubjects {
			// Never evict an active subject, but also never let client-controlled
			// cardinality turn this bounded cache into a route-wide fail-closed DoS.
			// Redis (when available), endpoint-global and IP fuses still run. Existing
			// tracked subjects keep their local budgets during Redis degradation.
			return Decision{
				Allowed:               true,
				Rule:                  rule,
				Limit:                 rule.Limit,
				LocalSubjectUntracked: true,
			}
		}
	}
	if !exists {
		expiresAt := now.Add(rule.Window)
		entry = &processCounter{expiresAt: expiresAt}
		bucket.entries[subject] = entry
		if bucket.nextExpiry.IsZero() || expiresAt.Before(bucket.nextExpiry) {
			bucket.nextExpiry = expiresAt
		}
	}
	entry.count += cost
	return Decision{
		Allowed: entry.count <= rule.Limit,
		Rule:    rule,
		Count:   entry.count,
		Limit:   rule.Limit,
	}
}

func (b *localScopeBucket) removeExpired(now time.Time) {
	if b == nil {
		return
	}
	var nextExpiry time.Time
	for subject, entry := range b.entries {
		if entry == nil || !now.Before(entry.expiresAt) {
			delete(b.entries, subject)
			continue
		}
		if nextExpiry.IsZero() || entry.expiresAt.Before(nextExpiry) {
			nextExpiry = entry.expiresAt
		}
	}
	b.nextExpiry = nextExpiry
}
