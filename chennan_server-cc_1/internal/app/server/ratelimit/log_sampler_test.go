package ratelimit

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeyedLogSamplerKeepsDimensionsIndependent(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	sampler := newKeyedLogSampler(10*time.Second, 8)

	if emit, suppressed := sampler.sample("login|ip", now); !emit || suppressed != 0 {
		t.Fatalf("first login sample = (%v, %d), want (true, 0)", emit, suppressed)
	}
	if emit, _ := sampler.sample("login|ip", now.Add(time.Second)); emit {
		t.Fatal("second event in the same key window was emitted")
	}
	if emit, suppressed := sampler.sample("sms|ip", now.Add(time.Second)); !emit || suppressed != 0 {
		t.Fatalf("independent SMS sample = (%v, %d), want (true, 0)", emit, suppressed)
	}

	if emit, suppressed := sampler.sample("login|ip", now.Add(10*time.Second)); !emit || suppressed != 1 {
		t.Fatalf("next login window sample = (%v, %d), want (true, 1)", emit, suppressed)
	}
}

func TestKeyedLogSamplerConcurrentWindowEmitsOnce(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	sampler := newKeyedLogSampler(time.Minute, 8)
	var emitted atomic.Int64
	var group sync.WaitGroup
	for i := 0; i < 100; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if emit, _ := sampler.sample("login|account", now); emit {
				emitted.Add(1)
			}
		}()
	}
	group.Wait()
	if got := emitted.Load(); got != 1 {
		t.Fatalf("concurrent emits = %d, want 1", got)
	}

	if emit, suppressed := sampler.sample("login|account", now.Add(time.Minute)); !emit || suppressed != 99 {
		t.Fatalf("next-window sample = (%v, %d), want (true, 99)", emit, suppressed)
	}
}

func TestKeyedLogSamplerBoundsKeys(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	sampler := newKeyedLogSampler(time.Minute, 3)
	for i := 0; i < 20; i++ {
		sampler.sample(fmt.Sprintf("key-%d", i), now.Add(time.Duration(i)*time.Millisecond))
	}
	if got := sampler.len(); got != 3 {
		t.Fatalf("sampler key count = %d, want 3", got)
	}
}

func TestSecurityLogSampleKeysExcludeRequestSubjects(t *testing.T) {
	if limitedLogSampleKey("login", "ip:minute") == limitedLogSampleKey("sms", "ip:minute") {
		t.Fatal("different actions unexpectedly share a limited-log sample key")
	}
	if limitedLogSampleKey("login", "ip:minute") == limitedLogSampleKey("login", "account:minute") {
		t.Fatal("different scopes unexpectedly share a limited-log sample key")
	}
	if overloadLogSampleKey("login", "cpu_gate_full") == overloadLogSampleKey("login", "rate_limit_check_failed") {
		t.Fatal("different overload reasons unexpectedly share a sample key")
	}
}
