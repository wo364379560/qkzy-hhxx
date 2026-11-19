package controller_chat

import (
	"context"
	"errors"
	"sync"
	"time"
)

type groupAdminRead struct {
	done   chan struct{}
	result *GroupAdminsResult
	err    error
}

// Merge only overlapping display reads. No stale permission cache is exposed
// to mutations, and failures/partial snapshots are never retained for reuse.
type groupAdminReadCoalescer struct {
	mu    sync.Mutex
	reads map[string]*groupAdminRead
}

var groupAdminReads groupAdminReadCoalescer

func (g *groupAdminReadCoalescer) load(ctx context.Context, key string, fetch func(context.Context) (*GroupAdminsResult, error)) (*GroupAdminsResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	g.mu.Lock()
	if g.reads == nil {
		g.reads = make(map[string]*groupAdminRead)
	}
	call := g.reads[key]
	if call == nil {
		if len(g.reads) >= 64 {
			g.mu.Unlock()
			return nil, errors.New("群管理员资料查询繁忙，请稍后重试")
		}
		call = &groupAdminRead{done: make(chan struct{})}
		g.reads[key] = call
		go func() {
			// One canceled browser request must not cancel every other waiter.
			readCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			call.result, call.err = fetch(readCtx)
			g.mu.Lock()
			delete(g.reads, key)
			close(call.done)
			g.mu.Unlock()
		}()
	}
	g.mu.Unlock()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-call.done:
		if call.err != nil {
			return nil, call.err
		}
		if call.result == nil {
			return nil, errors.New("群管理员资料为空")
		}
		result := *call.result
		result.Admins = append([]GroupAdminMemberResult{}, call.result.Admins...)
		if result.Owner != nil {
			owner := *result.Owner
			result.Owner = &owner
		}
		return &result, nil
	}
}
