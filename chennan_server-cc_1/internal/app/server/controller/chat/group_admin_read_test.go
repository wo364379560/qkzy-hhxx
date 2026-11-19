package controller_chat

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type observedWaitContext struct {
	context.Context
	observed chan struct{}
	once     sync.Once
}

func (c *observedWaitContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.observed) })
	return c.Context.Done()
}

func TestGroupAdminDisplayReadsCoalesceAndCancelIndependently(t *testing.T) {
	var loader groupAdminReadCoalescer
	var calls atomic.Int32
	release := make(chan struct{})
	fetch := func(ctx context.Context) (*GroupAdminsResult, error) {
		calls.Add(1)
		<-release
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return &GroupAdminsResult{Owner: &GroupAdminMemberResult{UserID: "owner"}, Admins: []GroupAdminMemberResult{{UserID: "admin"}}}, nil
	}
	canceled, cancel := context.WithCancel(context.Background())
	first := &observedWaitContext{Context: canceled, observed: make(chan struct{})}
	second := &observedWaitContext{Context: context.Background(), observed: make(chan struct{})}
	third := &observedWaitContext{Context: context.Background(), observed: make(chan struct{})}
	failed := make(chan error, 1)
	values := make(chan *GroupAdminsResult, 2)
	go func() { _, err := loader.load(first, "g", fetch); failed <- err }()
	<-first.observed
	for _, ctx := range []*observedWaitContext{second, third} {
		go func() {
			result, err := loader.load(ctx, "g", fetch)
			if err != nil {
				t.Error(err)
			}
			values <- result
		}()
		<-ctx.observed
	}
	cancel()
	if err := <-failed; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation=%v", err)
	}
	close(release)
	a, b := <-values, <-values
	if calls.Load() != 1 {
		t.Fatalf("upstream reads=%d, want 1", calls.Load())
	}
	if a == nil || b == nil {
		t.Fatal("remaining readers canceled")
	}
	a.Admins[0].UserID = "mutated"
	a.Owner.UserID = "mutated"
	if b.Admins[0].UserID != "admin" || b.Owner.UserID != "owner" {
		t.Fatal("callers share mutable response")
	}
	// A subsequent read gets fresh data; display results are not permission cache.
	if _, err := loader.load(context.Background(), "g", fetch); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal("old result was retained")
	}
}

func TestGroupAdminReadDoesNotCacheFailureOrPartialResult(t *testing.T) {
	var loader groupAdminReadCoalescer
	failure := errors.New("incomplete snapshot")
	result, err := loader.load(context.Background(), "g", func(context.Context) (*GroupAdminsResult, error) { return &GroupAdminsResult{}, failure })
	if !errors.Is(err, failure) || result != nil {
		t.Fatal("partial result escaped")
	}
	if _, err := loader.load(context.Background(), "g", func(context.Context) (*GroupAdminsResult, error) { return &GroupAdminsResult{}, nil }); err != nil {
		t.Fatal(err)
	}
}
