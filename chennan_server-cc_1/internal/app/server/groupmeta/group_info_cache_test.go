package groupmeta

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

func TestGroupMetadataCacheCoalescesConcurrentSends(t *testing.T) {
	c := newGroupInfoCache(16, time.Minute)
	var calls atomic.Int32
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			info, err := c.get(ctx, "group", func() (tencentim.GroupInfo, error) { calls.Add(1); return tencentim.GroupInfo{Name: "one"}, nil })
			if err != nil || info.Name != "one" {
				t.Errorf("lookup=%v %v", info, err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("100 concurrent sends fetched metadata %d times", calls.Load())
	}
}

func TestGroupMetadataInvalidationCannotResurrectOldFlight(t *testing.T) {
	c := newGroupInfoCache(2, time.Minute)
	started, release := make(chan struct{}), make(chan struct{})
	result := make(chan tencentim.GroupInfo, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() {
		info, _ := c.get(ctx, "group", func() (tencentim.GroupInfo, error) {
			close(started)
			<-release
			return tencentim.GroupInfo{Name: "old"}, nil
		})
		result <- info
	}()
	<-started
	c.invalidate("group")
	info, err := c.get(ctx, "group", func() (tencentim.GroupInfo, error) { return tencentim.GroupInfo{Name: "new"}, nil })
	if err != nil || info.Name != "new" {
		t.Fatalf("new lookup failed: %v", err)
	}
	close(release)
	if info = <-result; info.Name != "new" {
		t.Fatal("stale in-flight result survived invalidation")
	}
}

func TestGroupMetadataCacheBoundsSizeAndDoesNotCacheFailures(t *testing.T) {
	c := newGroupInfoCache(2, time.Minute)
	for i := 0; i < 8; i++ {
		_, err := c.get(context.Background(), fmt.Sprint(i), func() (tencentim.GroupInfo, error) { return tencentim.GroupInfo{}, nil })
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(c.entries) != 2 {
		t.Fatalf("entries=%d", len(c.entries))
	}
	var calls int
	for i := 0; i < 2; i++ {
		_, err := c.get(context.Background(), "failure", func() (tencentim.GroupInfo, error) { calls++; return tencentim.GroupInfo{}, errors.New("temporary") })
		if err == nil {
			t.Fatal("want failure")
		}
	}
	if calls != 2 {
		t.Fatal("temporary failure was cached")
	}
}

func TestGroupMetadataExpiresAndCallerCanCancel(t *testing.T) {
	c := newGroupInfoCache(2, time.Second)
	var now atomic.Int64
	now.Store(time.Now().Unix())
	c.now = func() time.Time { return time.Unix(now.Load(), 0) }
	var calls int
	load := func() (tencentim.GroupInfo, error) { calls++; return tencentim.GroupInfo{}, nil }
	c.get(context.Background(), "group", load)
	now.Add(2)
	c.get(context.Background(), "group", load)
	if calls != 2 {
		t.Fatal("expired metadata was reused")
	}
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := c.get(ctx, "slow", func() (tencentim.GroupInfo, error) { close(started); <-release; return tencentim.GroupInfo{}, nil })
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel=%v", err)
	}
}
