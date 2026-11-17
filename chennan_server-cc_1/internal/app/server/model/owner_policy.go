package model

// OwnerScope is resolved from current database rows, never from a client owner ID.
type OwnerScope struct {
	State   string `json:"state"`
	Reason  string `json:"reason_code,omitempty"`
	OwnerID uint   `json:"owner_id"`
	Role    string `json:"owner_role,omitempty"`
}

func ClassifyOwner(user *User, owner *AdminUser) OwnerScope {
	if user == nil || user.Status != "active" {
		return OwnerScope{State: "unavailable", Reason: "user_unavailable"}
	}
	s := OwnerScope{State: "unavailable", OwnerID: user.OwnerID}
	if user.OwnerID == 0 {
		s.State = "pool"
		return s
	}
	if owner == nil || owner.ID != user.OwnerID {
		s.Reason = "owner_missing"
		return s
	}
	s.Role = owner.Role
	if owner.Status != "active" {
		s.Reason = "owner_inactive"
		return s
	}
	switch owner.Role {
	case "super_admin":
		s.State = "pool"
	case "admin", "assistant":
		s.State = "assigned"
	default:
		s.Reason = "owner_role_unknown"
	}
	return s
}

// DiscoverOwnerPair is directional. Use the original applicant as from when
// accepting an application; confirmed relationships use SelectOwnerMove instead.
func DiscoverOwnerPair(from, to *User, a, b OwnerScope) string {
	if a.State == "unavailable" {
		return a.Reason
	}
	if b.State == "unavailable" {
		return b.Reason
	}
	if a.State == "pool" && b.State == "pool" || a.OwnerID == b.OwnerID {
		return ""
	}
	if a.State == "assigned" && b.State == "pool" {
		return ""
	}
	if a.State == "pool" && b.State == "assigned" {
		if to.GloballySearchable {
			return ""
		}
		return "not_globally_searchable"
	}
	return "cross_owner"
}

func SelectOwnerMove(a, b *User, sa, sb OwnerScope) (*User, uint, string) {
	if sa.State == "unavailable" {
		return nil, 0, sa.Reason
	}
	if sb.State == "unavailable" {
		return nil, 0, sb.Reason
	}
	if sa.State == "pool" && sb.State == "assigned" {
		return a, b.OwnerID, ""
	}
	if sb.State == "pool" && sa.State == "assigned" {
		return b, a.OwnerID, ""
	}
	if sa.State == "assigned" && sb.State == "assigned" && a.OwnerID != b.OwnerID {
		return nil, 0, "cross_owner"
	}
	return nil, 0, ""
}

type OwnerSummary struct {
	IMUserID      string `json:"im_user_id"`
	OwnerID       uint   `json:"owner_id"`
	OwnerRevision uint64 `json:"owner_revision"`
	IsAssigned    bool   `json:"is_assigned"`
	OwnerState    string `json:"owner_state"`
	OwnerRole     string `json:"owner_role,omitempty"`
	ReasonCode    string `json:"reason_code,omitempty"`
}

func SummarizeOwner(user *User, owner *AdminUser) OwnerSummary {
	scope := ClassifyOwner(user, owner)
	s := OwnerSummary{OwnerState: scope.State, OwnerRole: scope.Role, ReasonCode: scope.Reason}
	if user != nil {
		s.IMUserID = user.IMUserID
		s.OwnerID = user.OwnerID
		s.OwnerRevision = user.OwnerRevision
	}
	s.IsAssigned = scope.State == "assigned"
	if scope.State == "pool" {
		s.OwnerID = 0
	}
	return s
}

type OwnerAssignment struct {
	State          string       `json:"state"`
	ReasonCode     string       `json:"reason_code,omitempty"`
	RequestID      string       `json:"request_id"`
	Caller         OwnerSummary `json:"caller"`
	Target         OwnerSummary `json:"target"`
	AffectedGroups []uint       `json:"affected_groups"`
}
