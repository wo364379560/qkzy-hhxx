package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/ixayda/ix-Go/internal/app/server/model"

	"gorm.io/gorm"
)

const userOwnerGuard = `BEGIN
IF NOT (NEW.owner_id <=> OLD.owner_id) OR NOT (NEW.invite_code <=> OLD.invite_code) OR NOT (NEW.owner_revision <=> OLD.owner_revision) THEN
 IF COALESCE(@rui_owner_write,0) <> 1 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'rui_owner_transaction_required'; END IF;
 IF (NOT (NEW.owner_id <=> OLD.owner_id) OR NOT (NEW.invite_code <=> OLD.invite_code)) AND NEW.owner_revision <> OLD.owner_revision + 1 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'rui_owner_revision_required'; END IF;
 IF (NEW.owner_id <=> OLD.owner_id) AND (NEW.invite_code <=> OLD.invite_code) AND NEW.owner_revision <> OLD.owner_revision THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'rui_owner_noop_revision'; END IF;
END IF;
END`
const groupOwnerGuard = `BEGIN
IF NOT (NEW.owner_id <=> OLD.owner_id) OR NOT (NEW.created_by <=> OLD.created_by) OR NOT (NEW.owner_binding <=> OLD.owner_binding) OR NOT (NEW.owner_revision <=> OLD.owner_revision) OR NOT (NEW.mirror_revision <=> OLD.mirror_revision) THEN
 IF COALESCE(@rui_owner_write,0) <> 1 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'rui_group_owner_transaction_required'; END IF;
END IF;
END`

const groupInsertGuard = `BEGIN
DECLARE creator_owner BIGINT UNSIGNED DEFAULT NULL;
DECLARE creator_revision BIGINT UNSIGNED DEFAULT NULL;
IF COALESCE(@rui_owner_write,0) <> 1 AND NEW.owner_binding <> 'independent' THEN
 SELECT owner_id,owner_revision INTO creator_owner,creator_revision FROM users WHERE id=NEW.created_by FOR UPDATE;
 IF creator_owner IS NULL OR NOT (NEW.owner_id <=> creator_owner) OR NOT (NEW.owner_revision <=> creator_revision) THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'rui_stale_group_insert'; END IF;
END IF;
END`

// ownershipTransaction resets the connection-local capability before it returns
// to the pool. Triggers fence legacy full-row writers during staged rollouts.
func ownershipTransaction(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	return db.WithContext(ctx).Connection(func(conn *gorm.DB) (err error) {
		defer func() {
			// The request may already be cancelled/rolled back. Reset on the
			// reserved connection using a fresh context before pooling it again.
			cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			reset := conn.WithContext(cleanup).Exec("SET @rui_owner_write = 0").Error
			if reset != nil {
				if raw, ok := conn.Statement.ConnPool.(*sql.Conn); ok {
					_ = raw.Raw(func(any) error { return driver.ErrBadConn })
				}
			}
			if err == nil {
				err = reset
			}
		}()
		return conn.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SET @rui_owner_write = 1").Error; err != nil {
				return err
			}
			return fn(tx)
		})
	})
}

// EnsureOwnershipSchema is run after additive AutoMigrate and before any new
// ownership writer starts. Existing equivalent indexes are retained by columns.
func EnsureOwnershipSchema(db *gorm.DB) error {
	return withOwnershipSchemaLock(db, ensureOwnershipSchema)
}

// MigrateOwnershipSchema changes only the additive ownership columns, outbox and
// guards. It never backfills historical owners or changes authentication data.
func MigrateOwnershipSchema(db *gorm.DB) error {
	return withOwnershipSchemaLock(db, func(conn *gorm.DB) error {
		for _, entry := range []struct {
			row    any
			fields []string
		}{
			{&model.User{}, []string{"OwnerRevision"}},
			{&model.Group{}, []string{"OwnerBinding", "OwnerRevision", "MirrorRevision"}},
		} {
			for _, field := range entry.fields {
				if !conn.Migrator().HasColumn(entry.row, field) {
					if err := conn.Migrator().AddColumn(entry.row, field); err != nil {
						return err
					}
				}
			}
		}
		if err := conn.AutoMigrate(&model.FriendOwnerOperation{}); err != nil {
			return err
		}
		return ensureOwnershipSchema(conn)
	})
}

func withOwnershipSchemaLock(db *gorm.DB, fn func(*gorm.DB) error) error {
	return db.Connection(func(conn *gorm.DB) error {
		var acquired int
		if err := conn.Raw("SELECT GET_LOCK(CONCAT('rui_owner_schema:',DATABASE()),15)").Scan(&acquired).Error; err != nil {
			return err
		}
		if acquired != 1 {
			return fmt.Errorf("ownership schema migration is busy")
		}
		defer conn.Exec("SELECT RELEASE_LOCK(CONCAT('rui_owner_schema:',DATABASE()))")
		return fn(conn)
	})
}

func ensureOwnershipSchema(db *gorm.DB) error {
	for _, entry := range []struct{ name, table, timing, body string }{{"rui_user_owner_guard_v1", "users", "UPDATE", userOwnerGuard}, {"rui_group_owner_guard_v1", "groups", "UPDATE", groupOwnerGuard}, {"rui_group_insert_guard_v1", "groups", "INSERT", groupInsertGuard}} {
		var existing struct {
			Body string `gorm:"column:ACTION_STATEMENT"`
		}
		if err := db.Raw("SELECT ACTION_STATEMENT FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME=?", entry.name).Scan(&existing).Error; err != nil {
			return err
		}
		normalize := func(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
		if existing.Body != "" {
			if normalize(existing.Body) != normalize(entry.body) {
				return fmt.Errorf("ownership guard definition requires review: %s", entry.name)
			}
			continue
		}
		if err := db.Exec("CREATE TRIGGER " + entry.name + " BEFORE " + entry.timing + " ON `" + entry.table + "` FOR EACH ROW " + entry.body).Error; err != nil {
			return err
		}
	}
	for _, index := range []struct {
		table, name string
		columns     []string
	}{
		{"groups", "idx_rui_groups_creator_status", []string{"created_by", "status"}},
		{"users", "idx_rui_users_owner_status", []string{"owner_id", "status"}},
		{"invite_codes", "idx_rui_invite_owner_id", []string{"owner_id", "id"}},
	} {
		var rows []struct {
			Name   string `gorm:"column:INDEX_NAME"`
			Column string `gorm:"column:COLUMN_NAME"`
		}
		if err := db.Raw("SELECT INDEX_NAME,COLUMN_NAME FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? ORDER BY INDEX_NAME,SEQ_IN_INDEX", index.table).Scan(&rows).Error; err != nil {
			return err
		}
		byName := map[string][]string{}
		for _, row := range rows {
			byName[row.Name] = append(byName[row.Name], row.Column)
		}
		covered := false
		for _, cols := range byName {
			if len(cols) >= len(index.columns) && strings.Join(cols[:len(index.columns)], ",") == strings.Join(index.columns, ",") {
				covered = true
			}
		}
		if !covered {
			if err := db.Exec("CREATE INDEX " + index.name + " ON `" + index.table + "` (" + strings.Join(index.columns, ",") + ")").Error; err != nil {
				return err
			}
		}
	}
	return nil
}
