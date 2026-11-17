package model

import "testing"

func TestOwnerPolicyStages(t *testing.T) {
	super := &AdminUser{ID: 1, Role: "super_admin", Status: "active"}
	a := &AdminUser{ID: 10, Role: "admin", Status: "active"}
	b := &AdminUser{ID: 20, Role: "assistant", Status: "active"}
	pool := &User{ID: 1, OwnerID: 1, Status: "active"}
	assigned := &User{ID: 2, OwnerID: 10, Status: "active"}
	other := &User{ID: 3, OwnerID: 20, Status: "active", GloballySearchable: true}
	if DiscoverOwnerPair(pool, assigned, ClassifyOwner(pool, super), ClassifyOwner(assigned, a)) != "not_globally_searchable" {
		t.Fatal("pool discovery must respect the existing switch")
	}
	if DiscoverOwnerPair(assigned, pool, ClassifyOwner(assigned, a), ClassifyOwner(pool, super)) != "" {
		t.Fatal("original assigned applicant can add pool user")
	}
	// Accepting client has the reverse direction, but post-confirmation allocation
	// must not reinterpret it as a new pool-to-hidden-account search.
	move, owner, reason := SelectOwnerMove(pool, assigned, ClassifyOwner(pool, super), ClassifyOwner(assigned, a))
	if move != pool || owner != 10 || reason != "" {
		t.Fatal("confirmed friendship lost the original hidden-account intent")
	}
	if DiscoverOwnerPair(assigned, other, ClassifyOwner(assigned, a), ClassifyOwner(other, b)) != "cross_owner" {
		t.Fatal("global visibility must not enable cross-owner migration")
	}
	for _, owner := range []*AdminUser{nil, {ID: 1, Role: "super_admin", Status: "banned"}, {ID: 1, Role: "unknown", Status: "active"}} {
		scope := ClassifyOwner(pool, owner)
		if scope.State != "unavailable" || scope.Reason == "" {
			t.Fatal("invalid owner became pool")
		}
	}
	if ClassifyOwner(&User{OwnerID: 0, Status: "active"}, nil).State != "pool" {
		t.Fatal("zero owner is the default pool")
	}
	if ClassifyOwner(&User{OwnerID: 10, Status: "banned"}, a).State != "unavailable" {
		t.Fatal("banned user passed policy")
	}
}
