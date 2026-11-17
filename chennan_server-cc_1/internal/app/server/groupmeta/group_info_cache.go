package groupmeta

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
)

// Cache remote metadata only. Local owner, risk state and policy are read for
// every send; a cached allow/deny decision must never bypass a new policy.
type groupInfoEntry struct {
	done    chan struct{}
	info    tencentim.GroupInfo
	err     error
	expires time.Time
}
type groupInfoCache struct {
	mu       sync.Mutex
	entries  map[string]*groupInfoEntry
	capacity int
	ttl      time.Duration
	now      func() time.Time
}

func newGroupInfoCache(capacity int, ttl time.Duration) *groupInfoCache {
	return &groupInfoCache{entries: make(map[string]*groupInfoEntry), capacity: capacity, ttl: ttl, now: time.Now}
}

var beforeSendGroupInfo = newGroupInfoCache(2048, 3*time.Second)

func (c *groupInfoCache) invalidate(id string) {
	c.mu.Lock()
	delete(c.entries, id)
	c.mu.Unlock()
}
func Invalidate(groupID string) { beforeSendGroupInfo.invalidate(groupID) }

func (c *groupInfoCache) get(ctx context.Context, id string, load func() (tencentim.GroupInfo, error)) (tencentim.GroupInfo, error) {
	for {
		c.mu.Lock()
		entry := c.entries[id]
		if entry != nil && !entry.expires.IsZero() && !entry.expires.After(c.now()) {
			delete(c.entries, id)
			entry = nil
		}
		if entry == nil {
			if len(c.entries) >= c.capacity {
				var oldestID string
				var oldest time.Time
				for key, item := range c.entries {
					if !item.expires.IsZero() && (oldestID == "" || item.expires.Before(oldest)) {
						oldestID = key
						oldest = item.expires
					}
				}
				if oldestID == "" {
					c.mu.Unlock()
					return tencentim.GroupInfo{}, errors.New("group metadata capacity busy")
				}
				delete(c.entries, oldestID)
			}
			entry = &groupInfoEntry{done: make(chan struct{})}
			c.entries[id] = entry
			go func(pending *groupInfoEntry) {
				info, err := load()
				c.mu.Lock()
				pending.info = info
				pending.err = err
				pending.expires = c.now().Add(c.ttl)
				if err != nil && c.entries[id] == pending {
					delete(c.entries, id)
				}
				close(pending.done)
				c.mu.Unlock()
			}(entry)
		}
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return tencentim.GroupInfo{}, ctx.Err()
		case <-entry.done:
			c.mu.Lock()
			current := c.entries[id] == entry
			c.mu.Unlock()
			if entry.err != nil {
				return tencentim.GroupInfo{}, entry.err
			}
			if current {
				return entry.info, nil
			}
			// An event invalidated this flight: never repopulate stale metadata.
		}
	}
}

func Load(ctx context.Context, groupID string, load func() (tencentim.GroupInfo, error)) (tencentim.GroupInfo, error) {
	return beforeSendGroupInfo.get(ctx, groupID, load)
}
