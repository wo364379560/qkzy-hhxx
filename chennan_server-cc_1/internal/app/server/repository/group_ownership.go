package repository

import (
	"context"
	"errors"

	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewGroupRepositoryWithDB(db *gorm.DB) *GroupRepository { return &GroupRepository{db: db} }

// SaveMirror consumes verified Tencent metadata, but derives the business owner
// again under user locks. A stale OwnerUser snapshot is never authoritative.
func (r *GroupRepository) SaveMirror(ctx context.Context, g *model.Group) error {
	if g == nil || g.CreatedBy == 0 {
		return OwnershipError{"creator_unavailable"}
	}
	var existing model.Group
	err := r.db.WithContext(ctx).Where("im_group_id = ?", g.IMGroupID).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && existing.OwnerBinding == "independent" {
		// Admin-created groups use an admin ID for CreatedBy. Never reinterpret
		// that ID as a business user when refreshing their Tencent metadata.
		existing.Name, existing.GroupType = g.Name, g.GroupType
		if err := r.Update(&existing); err != nil {
			return err
		}
		*g = existing
		return nil
	}
	ids := []uint{g.CreatedBy}
	if err == nil && existing.CreatedBy > 0 && existing.OwnerBinding != "independent" {
		ids = append(ids, existing.CreatedBy)
	}
	return ownershipTransaction(ctx, r.db, func(tx *gorm.DB) error {
		s, err := lockOwnerRows(tx, ids, nil)
		if err != nil {
			return err
		}
		creator := s.Users[g.CreatedBy]
		scope := s.Scope(creator)
		if scope.State == "unavailable" {
			return OwnershipError{scope.Reason}
		}
		var current model.Group
		found := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("im_group_id = ?", g.IMGroupID).First(&current).Error
		if errors.Is(found, gorm.ErrRecordNotFound) {
			g.OwnerID = creator.OwnerID
			g.OwnerRevision = creator.OwnerRevision
			g.OwnerBinding = "follow_user"
			return tx.Create(g).Error
		}
		if found != nil {
			return found
		}
		fields := map[string]any{"name": g.Name, "group_type": g.GroupType}
		if current.OwnerBinding != "independent" {
			if current.OwnerBinding != "follow_user" {
				var count int64
				if err := tx.Model(&model.AdminUser{}).Where("id = ?", current.CreatedBy).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return OwnershipError{"group_creator_ambiguous"}
				}
			}
			if current.CreatedBy != g.CreatedBy && current.MirrorRevision != g.MirrorRevision {
				return OwnershipError{"stale_group_mirror"}
			}
			previous := s.Users[current.CreatedBy]
			if previous == nil {
				return OwnershipError{"creator_changed"}
			}
			if current.OwnerBinding != "follow_user" && current.OwnerID != previous.OwnerID {
				return OwnershipError{"group_owner_conflict"}
			}
			fields["owner_id"] = creator.OwnerID
			fields["created_by"] = creator.ID
			fields["owner_revision"] = creator.OwnerRevision
			fields["owner_binding"] = "follow_user"
			fields["mirror_revision"] = gorm.Expr("mirror_revision + 1")
		}
		if err := tx.Model(&current).Updates(fields).Error; err != nil {
			return err
		}
		if err := tx.First(&current, current.ID).Error; err != nil {
			return err
		}
		*g = current
		return nil
	})
}
