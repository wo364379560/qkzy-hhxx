package ratelimit

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLocalLimiterAllowsExactlyLimitWithinWindow(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	limiter := NewLocalLimiter(LocalOptions{
		Limit:       3,
		Window:      time.Minute,
		MaxSubjects: 16,
		Now:         func() time.Time { return now },
	})

	for i := 1; i <= 3; i++ {
		decision := limiter.Check("client-a", 1)
		if !decision.Allowed {
			t.Fatalf("request %d unexpectedly denied: %+v", i, decision)
		}
	}

	decision := limiter.Check("client-a", 1)
	if decision.Allowed {
		t.Fatalf("request above limit unexpectedly allowed: %+v", decision)
	}
}

func TestLocalLimiterChargesCostAndNormalizesNonPositiveCost(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	limiter := NewLocalLimiter(LocalOptions{
		Limit:       5,
		Window:      time.Minute,
		MaxSubjects: 16,
		Now:         func() time.Time { return now },
	})

	if decision := limiter.Check("costed", 2); !decision.Allowed {
		t.Fatalf("first cost=2 request denied: %+v", decision)
	}
	if decision := limiter.Check("costed", 2); !decision.Allowed {
		t.Fatalf("second cost=2 request denied: %+v", decision)
	}
	if decision := limiter.Check("costed", 0); !decision.Allowed {
		t.Fatalf("non-positive cost should be normalized to one: %+v", decision)
	}
	if decision := limiter.Check("costed", -10); decision.Allowed {
		t.Fatalf("request above accumulated limit unexpectedly allowed: %+v", decision)
	}
}

func TestLocalLimiterResetsAtWindowBoundary(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	window := 30 * time.Second
	limiter := NewLocalLimiter(LocalOptions{
		Limit:       1,
		Window:      window,
		MaxSubjects: 16,
		Now:         func() time.Time { return now },
	})

	if decision := limiter.Check("client-a", 1); !decision.Allowed {
		t.Fatalf("first request denied: %+v", decision)
	}
	if decision := limiter.Check("client-a", 1); decision.Allowed {
		t.Fatalf("second request in same window unexpectedly allowed: %+v", decision)
	}

	now = now.Add(window)
	if decision := limiter.Check("client-a", 1); !decision.Allowed {
		t.Fatalf("request at new window boundary denied: %+v", decision)
	}
}

func TestLocalLimiterKeepsSubjectsIndependent(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	limiter := NewLocalLimiter(LocalOptions{
		Limit:       1,
		Window:      time.Minute,
		MaxSubjects: 16,
		Now:         func() time.Time { return now },
	})

	if decision := limiter.Check("client-a", 1); !decision.Allowed {
		t.Fatalf("client-a first request denied: %+v", decision)
	}
	if decision := limiter.Check("client-a", 1); decision.Allowed {
		t.Fatalf("client-a second request unexpectedly allowed: %+v", decision)
	}
	if decision := limiter.Check("client-b", 1); !decision.Allowed {
		t.Fatalf("client-b should have an independent budget: %+v", decision)
	}
}

func TestLocalLimiterConcurrentChecksDoNotExceedLimit(t *testing.T) {
	const (
		limit    = 32
		requests = 512
	)
	limiter := NewLocalLimiter(LocalOptions{
		Limit:       limit,
		Window:      time.Hour,
		MaxSubjects: 64,
		Now:         time.Now,
	})

	start := make(chan struct{})
	var allowed atomic.Int64
	var wg sync.WaitGroup
	wg.Add(requests)
	for i := 0; i < requests; i++ {
		go func() {
			defer wg.Done()
			<-start
			if limiter.Check("same-client", 1).Allowed {
				allowed.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := allowed.Load(); got != limit {
		t.Fatalf("allowed requests = %d, want exactly %d", got, limit)
	}
	if got := limiter.Len(); got != 1 {
		t.Fatalf("tracked subjects = %d, want 1", got)
	}
}

func TestLocalLimiterSubjectCardinalityStaysBoundedConcurrently(t *testing.T) {
	const (
		maxSubjects = 32
		subjects    = 1024
	)
	limiter := NewLocalLimiter(LocalOptions{
		Limit:       10,
		Window:      time.Hour,
		MaxSubjects: maxSubjects,
		Now:         time.Now,
	})

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(subjects)
	for i := 0; i < subjects; i++ {
		subject := fmt.Sprintf("rotating-client-%d", i)
		go func() {
			defer wg.Done()
			<-start
			_ = limiter.Check(subject, 1)
		}()
	}
	close(start)
	wg.Wait()

	if got := limiter.Len(); got > maxSubjects {
		t.Fatalf("tracked subjects = %d, exceeds MaxSubjects %d", got, maxSubjects)
	}
}

func TestLocalLimiterRejectsNewSubjectInsteadOfEvictingActiveBudget(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	limiter := NewLocalLimiter(LocalOptions{
		Limit:       1,
		Window:      time.Hour,
		MaxSubjects: 2,
		Now:         func() time.Time { return now },
	})

	if decision := limiter.Check("tracked-a", 1); !decision.Allowed {
		t.Fatalf("tracked-a first request denied: %+v", decision)
	}
	if decision := limiter.Check("tracked-b", 1); !decision.Allowed {
		t.Fatalf("tracked-b first request denied: %+v", decision)
	}
	if decision := limiter.Check("rotating-attacker", 1); decision.Allowed {
		t.Fatalf("new subject was allowed after the active key budget was full: %+v", decision)
	}
	if decision := limiter.Check("tracked-a", 1); decision.Allowed {
		t.Fatalf("active tracked subject was evicted and regained budget: %+v", decision)
	}
	if got := limiter.Len(); got != 2 {
		t.Fatalf("tracked subjects = %d, want 2", got)
	}
}

func TestProcessLimiterLeavesNewSubjectToRedisInsteadOfEvictingActiveBudget(t *testing.T) {
	store := localRuleStore{
		buckets:             make(map[string]*localScopeBucket),
		maxScopes:           4,
		maxSubjectsPerScope: 2,
	}
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	deviceRule := Rule{Scope: "device", Limit: 1, Window: time.Hour}
	accountRule := Rule{Scope: "account", Limit: 1, Window: time.Hour}

	if !store.check("device-a", 1, deviceRule, now).Allowed ||
		!store.check("device-b", 1, deviceRule, now).Allowed {
		t.Fatal("initial device subjects should be allowed")
	}
	if !store.check("account-a", 1, accountRule, now).Allowed {
		t.Fatal("initial account subject should be allowed")
	}
	if decision := store.check("device-c", 1, deviceRule, now); !decision.Allowed || !decision.LocalSubjectUntracked {
		t.Fatalf("new device subject should remain untracked for Redis/global safeguards: %+v", decision)
	}
	if store.check("device-a", 1, deviceRule, now).Allowed {
		t.Fatal("device churn evicted an active device budget")
	}
	if store.check("account-a", 1, accountRule, now).Allowed {
		t.Fatal("device churn reset an unrelated account budget")
	}
}

func TestProcessLimiterChurnCannotResetVictimBudget(t *testing.T) {
	store := localRuleStore{
		buckets:             make(map[string]*localScopeBucket),
		maxScopes:           2,
		maxSubjectsPerScope: 2,
	}
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	rule := Rule{Scope: "account", Limit: 1, Window: time.Hour}

	if !store.check("victim", 1, rule, now).Allowed ||
		!store.check("active-peer", 1, rule, now).Allowed {
		t.Fatal("initial account subjects should be allowed")
	}
	for i := 0; i < 4096; i++ {
		decision := store.check(fmt.Sprintf("rotating-account-%d", i), 1, rule, now)
		if !decision.Allowed || !decision.LocalSubjectUntracked {
			t.Fatalf("rotating account %d did not take the bounded untracked path: %+v", i, decision)
		}
	}
	if decision := store.check("victim", 1, rule, now); decision.Allowed {
		t.Fatalf("victim regained budget after account churn: %+v", decision)
	}
	if got := len(store.buckets["account"].entries); got != 2 {
		t.Fatalf("tracked account subjects = %d, want 2", got)
	}
}

func TestProcessLimiterReclaimsExpiredSubjectBeforeRejectingNew(t *testing.T) {
	store := localRuleStore{
		buckets:             make(map[string]*localScopeBucket),
		maxScopes:           2,
		maxSubjectsPerScope: 2,
	}
	start := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	rule := Rule{Scope: "account", Limit: 1, Window: time.Hour}

	if !store.check("expires-first", 1, rule, start).Allowed {
		t.Fatal("first subject should be allowed")
	}
	activeAtBoundary := start.Add(30 * time.Minute)
	if !store.check("still-active", 1, rule, activeAtBoundary).Allowed {
		t.Fatal("second subject should be allowed")
	}
	boundary := start.Add(time.Hour)
	if decision := store.check("newcomer", 1, rule, boundary); !decision.Allowed {
		t.Fatalf("new subject was denied even though an expired slot was reclaimable: %+v", decision)
	}
	if decision := store.check("still-active", 1, rule, boundary); decision.Allowed {
		t.Fatalf("active subject was evicted while reclaiming expired capacity: %+v", decision)
	}

	bucket := store.buckets["account"]
	if got := len(bucket.entries); got != 2 {
		t.Fatalf("tracked account subjects = %d, want 2", got)
	}
	if _, exists := bucket.entries["expires-first"]; exists {
		t.Fatal("expired subject was not reclaimed")
	}
	if _, exists := bucket.entries["newcomer"]; !exists {
		t.Fatal("new subject was not stored in the reclaimed slot")
	}
}

func TestProcessLimiterConcurrentChurnRemainsBoundedAndPreservesBudgets(t *testing.T) {
	const (
		maxSubjects = 32
		requests    = 1024
	)
	store := localRuleStore{
		buckets:             make(map[string]*localScopeBucket),
		maxScopes:           2,
		maxSubjectsPerScope: maxSubjects,
	}
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	rule := Rule{Scope: "account", Limit: 1, Window: time.Hour}
	for i := 0; i < maxSubjects; i++ {
		if decision := store.check(fmt.Sprintf("tracked-%d", i), 1, rule, now); !decision.Allowed {
			t.Fatalf("initial subject %d denied: %+v", i, decision)
		}
	}

	start := make(chan struct{})
	var untracked atomic.Int64
	var wg sync.WaitGroup
	wg.Add(requests)
	for i := 0; i < requests; i++ {
		subject := fmt.Sprintf("concurrent-rotating-%d", i)
		go func() {
			defer wg.Done()
			<-start
			decision := store.check(subject, 1, rule, now)
			if decision.Allowed && decision.LocalSubjectUntracked {
				untracked.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := untracked.Load(); got != requests {
		t.Fatalf("bounded untracked subjects = %d, want %d", got, requests)
	}
	if decision := store.check("tracked-0", 1, rule, now); decision.Allowed {
		t.Fatalf("tracked subject regained budget after concurrent churn: %+v", decision)
	}
	if got := len(store.buckets["account"].entries); got != maxSubjects {
		t.Fatalf("tracked account subjects = %d, want %d", got, maxSubjects)
	}
}

func TestProcessLimiterScopeCardinalityIsBounded(t *testing.T) {
	store := localRuleStore{
		buckets:             make(map[string]*localScopeBucket),
		maxScopes:           1,
		maxSubjectsPerScope: 2,
	}
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	first := Rule{Scope: "known", Limit: 10, Window: time.Hour}
	second := Rule{Scope: "unexpected", Limit: 10, Window: time.Hour}
	if !store.check("a", 1, first, now).Allowed {
		t.Fatal("first compiled scope should be allowed")
	}
	if store.check("b", 1, second, now).Allowed {
		t.Fatal("scope allocation above the configured maximum should fail closed")
	}
}
