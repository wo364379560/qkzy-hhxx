package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	mysqlconfig "github.com/go-sql-driver/mysql"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ownershipTestDB(t *testing.T, legacy ...bool) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("RUI_OWNER_MYSQL_DSN")
	if dsn == "" {
		t.Skip("RUI_OWNER_MYSQL_DSN must name a disposable loopback MySQL database")
	}
	cfg, err := mysqlconfig.ParseDSN(dsn)
	if err != nil || cfg.Net != "tcp" || !strings.HasPrefix(cfg.Addr, "127.0.0.1:") || !strings.HasPrefix(cfg.DBName, "rui_owner_test") {
		t.Fatal("refusing a non-isolated ownership test database")
	}
	cfg.DBName = ""
	admin, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated MySQL unavailable")
	}
	name := "rui_owner_test_" + OwnershipID()[:12]
	if err := admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4").Error; err != nil {
		t.Fatal(err)
	}
	cfg.DBName = name
	db, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated schema unavailable")
	}
	t.Cleanup(func() {
		conn, _ := db.DB()
		_ = conn.Close()
		_ = admin.Exec("DROP DATABASE " + name).Error
		c, _ := admin.DB()
		_ = c.Close()
	})
	if len(legacy) > 0 && legacy[0] {
		return db
	}
	if err := db.AutoMigrate(&model.AdminUser{}, &model.User{}, &model.Group{}, &model.InviteCode{}, &model.FriendOwnerOperation{}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureOwnershipSchema(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func ownershipFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, a := range []model.AdminUser{{ID: 100, Username: "super", DisplayName: "Super", Role: "super_admin", Status: "active"}, {ID: 110, Username: "space-a", DisplayName: "A", Role: "admin", Status: "active"}, {ID: 120, Username: "space-b", DisplayName: "B", Role: "admin", Status: "active"}} {
		if err := db.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []model.User{{ID: 201, OwnerID: 100}, {ID: 202, OwnerID: 110}, {ID: 203, OwnerID: 120}} {
		u.BizID = int64(u.ID)
		u.Phone = fmt.Sprintf("fixture-%d", u.ID)
		u.IMUserID = fmt.Sprintf("u_%d", u.ID)
		u.Nickname = u.IMUserID
		u.Status = "active"
		u.AuthVersion = 7
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, code := range []model.InviteCode{{Code: "CODE-A", OwnerID: 110, Status: "active"}, {Code: "CODE-B", OwnerID: 120, Status: "active"}} {
		if err := db.Create(&code).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func claimOwnerPair(t *testing.T, r *OwnershipRepository, event, from, to string) *model.FriendOwnerOperation {
	t.Helper()
	op, err := r.Enqueue(context.Background(), "client", event, from, to)
	if err != nil {
		t.Fatal(err)
	}
	op, err = r.Claim(context.Background(), op.ID)
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func TestOwnerTransactionsMySQL(t *testing.T) {
	ctx := context.Background()
	t.Run("cancelled_transaction_never_leaks_legacy_write_capability", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		pool, _ := db.DB()
		pool.SetMaxOpenConns(1)
		pool.SetMaxIdleConns(1)
		cancelled, cancel := context.WithCancel(ctx)
		_ = ownershipTransaction(cancelled, db, func(tx *gorm.DB) error { cancel(); return context.Canceled })
		var capability int
		if err := db.Raw("SELECT COALESCE(@rui_owner_write,0)").Scan(&capability).Error; err != nil {
			t.Fatal(err)
		}
		if capability != 0 {
			t.Fatal("cancelled transaction leaked connection capability")
		}
		if err := db.Model(&model.User{}).Where("id=201").Update("owner_id", 120).Error; err == nil {
			t.Fatal("pooled connection bypassed legacy guard")
		}
	})
	t.Run("legacy_schema_additive_migration_twice", func(t *testing.T) {
		db := ownershipTestDB(t, true)
		for _, statement := range []string{
			"CREATE TABLE users (id BIGINT UNSIGNED PRIMARY KEY,owner_id BIGINT UNSIGNED,status VARCHAR(16),invite_code VARCHAR(32),auth_version INT)",
			"CREATE TABLE `groups` (id BIGINT UNSIGNED PRIMARY KEY,owner_id BIGINT UNSIGNED,created_by BIGINT UNSIGNED,status VARCHAR(16))",
			"CREATE TABLE invite_codes (id BIGINT UNSIGNED PRIMARY KEY,owner_id BIGINT UNSIGNED)",
			"INSERT INTO users VALUES(1,35,'active','OLD-CODE',17)",
			"INSERT INTO `groups` VALUES(2,35,1,'active')",
		} {
			if err := db.Exec(statement).Error; err != nil {
				t.Fatal(err)
			}
		}
		for i := 0; i < 2; i++ {
			if err := MigrateOwnershipSchema(db); err != nil {
				t.Fatal(err)
			}
		}
		var user struct {
			OwnerID       uint
			OwnerRevision uint64
			InviteCode    string
			AuthVersion   int
		}
		if err := db.Table("users").First(&user).Error; err != nil {
			t.Fatal(err)
		}
		if user.OwnerID != 35 || user.OwnerRevision != 0 || user.InviteCode != "OLD-CODE" || user.AuthVersion != 17 {
			t.Fatal("migration rewrote historical tuple or credentials")
		}
		var group struct {
			OwnerID       uint
			OwnerRevision uint64
			OwnerBinding  string
		}
		db.Table("groups").First(&group)
		if group.OwnerID != 35 || group.OwnerRevision != 0 || group.OwnerBinding != "legacy" {
			t.Fatal("migration invented a following group")
		}
	})
	t.Run("bilateral_confirmed_direction_and_idempotence", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		g := model.Group{ID: 301, BizID: 301, IMGroupID: "g_301", CreatedBy: 201, OwnerID: 100, Status: "active"}
		if err := db.Create(&g).Error; err != nil {
			t.Fatal(err)
		}
		op := claimOwnerPair(t, r, "accept-hidden", "u_201", "u_202")
		result, err := r.ApplyConfirmed(ctx, op, []string{"u_202"})
		if err != nil {
			t.Fatal(err)
		}
		if result.State != "migrated" || result.Caller.OwnerID != 110 || result.Caller.OwnerRevision != 1 {
			t.Fatal("wrong default-pool direction or version")
		}
		var u model.User
		db.First(&u, 201)
		db.First(&g, 301)
		if u.InviteCode != "CODE-A" || u.AuthVersion != 7 || g.OwnerID != 110 || g.OwnerRevision != 1 {
			t.Fatal("user/invitation/group/auth atomic contract broken")
		}
		var saved model.FriendOwnerOperation
		db.First(&saved, op.ID)
		var before, after struct {
			Users []struct {
				InviteCode string `json:"invite_code"`
			}
			Groups []struct {
				OwnerID uint `json:"owner_id"`
			}
		}
		if json.Unmarshal([]byte(saved.BeforeJSON), &before) != nil || json.Unmarshal([]byte(saved.AfterJSON), &after) != nil || len(before.Groups) != 1 || before.Groups[0].OwnerID != 100 || after.Groups[0].OwnerID != 110 || before.Users[0].InviteCode != "" || after.Users[0].InviteCode != "CODE-A" {
			t.Fatal("atomic ownership audit lost invitation or group history")
		}
		duplicate, err := r.Enqueue(ctx, "client", "accept-hidden", "u_201", "u_202")
		if err != nil || duplicate.ID != op.ID {
			t.Fatal("stable event was duplicated")
		}
		next := claimOwnerPair(t, r, "second-client", "u_202", "u_201")
		result, err = r.ApplyConfirmed(ctx, next, []string{"u_201"})
		if err != nil || result.State != "unchanged" {
			t.Fatal("replay is not idempotent")
		}
		db.First(&u, 201)
		if u.OwnerRevision != 1 {
			t.Fatal("revision increased on no-op")
		}
	})
	t.Run("owner_validity_rechecked_without_user_revision_change", func(t *testing.T) {
		for _, kind := range []string{"banned", "missing", "unknown-role"} {
			t.Run(kind, func(t *testing.T) {
				db := ownershipTestDB(t)
				ownershipFixtures(t, db)
				r := &OwnershipRepository{DB: db}
				op := claimOwnerPair(t, r, "owner-changed", "u_201", "u_202")
				if kind == "missing" {
					db.Delete(&model.AdminUser{}, 110)
				} else if kind == "banned" {
					db.Model(&model.AdminUser{}).Where("id=110").Update("status", "banned")
				} else {
					db.Exec("ALTER TABLE admin_users MODIFY role VARCHAR(32)")
					db.Model(&model.AdminUser{}).Where("id=110").Update("role", "unrecognized")
				}
				result, err := r.ApplyConfirmed(ctx, op, []string{"u_202"})
				if err != nil || result.State != "blocked" {
					t.Fatal("invalid owner was treated as a pool")
				}
				var user, target model.User
				db.First(&user, 201)
				db.First(&target, 202)
				if user.OwnerID != 100 || user.OwnerRevision != 0 || target.OwnerRevision != 0 {
					t.Fatal("owner validity change modified user revision or migrated a pool")
				}
			})
		}
	})
	t.Run("friend_and_legal_admin_transfer_serialize", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		db.Create(&model.Group{ID: 301, BizID: 301, IMGroupID: "admin-race", OwnerID: 100, CreatedBy: 201, Status: "active"})
		op := claimOwnerPair(t, r, "admin-race", "u_201", "u_202")
		start := make(chan struct{})
		errs := make(chan error, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); <-start; _, err := r.ApplyConfirmed(ctx, op, []string{"u_202"}); errs <- err }()
		go func() { defer wg.Done(); <-start; _, err := r.Transfer(ctx, 100, 201, 120); errs <- err }()
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		var u model.User
		var g model.Group
		db.First(&u, 201)
		db.First(&g, 301)
		if u.OwnerID != 120 || g.OwnerID != 120 || u.OwnerRevision != g.OwnerRevision || u.OwnerRevision < 1 || u.OwnerRevision > 2 || u.AuthVersion != 7 {
			t.Fatal("admin transfer and friendship split the protected tuple")
		}
	})
	t.Run("two_operating_spaces_cannot_overwrite_each_other", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		jobs := []*model.FriendOwnerOperation{claimOwnerPair(t, r, "a", "u_202", "u_201"), claimOwnerPair(t, r, "b", "u_203", "u_201")}
		start := make(chan struct{})
		var wg sync.WaitGroup
		states := make(chan string, 2)
		errs := make(chan error, 2)
		for _, job := range jobs {
			wg.Add(1)
			go func(j *model.FriendOwnerOperation) {
				defer wg.Done()
				<-start
				result, err := r.ApplyConfirmed(ctx, j, []string{"u_201"})
				if err != nil {
					errs <- err
					return
				}
				states <- result.State
			}(job)
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Fatal(err)
		}
		close(states)
		wins, conflicts := 0, 0
		for state := range states {
			if state == "migrated" {
				wins++
			}
			if state == "conflict" {
				conflicts++
			}
		}
		var u model.User
		db.First(&u, 201)
		if wins != 1 || conflicts != 1 || u.OwnerRevision != 1 {
			t.Fatal("parallel assignment did not have one stable winner")
		}
	})
	t.Run("group_failure_rolls_back_all_assignment_fields", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		db.Create(&model.Group{ID: 301, BizID: 301, IMGroupID: "g_301", OwnerID: 100, CreatedBy: 201, Status: "active"})
		if err := db.Exec("CREATE TRIGGER owner_test_group_failure BEFORE UPDATE ON `groups` FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected_group_failure'").Error; err != nil {
			t.Fatal(err)
		}
		op := claimOwnerPair(t, r, "rollback", "u_201", "u_202")
		if _, err := r.ApplyConfirmed(ctx, op, []string{"u_202"}); err == nil {
			t.Fatal("injected group write succeeded")
		}
		var u model.User
		var g model.Group
		db.First(&u, 201)
		db.First(&g, 301)
		if u.OwnerID != 100 || u.InviteCode != "" || u.OwnerRevision != 0 || g.OwnerID != 100 {
			t.Fatal("partial assignment escaped rollback")
		}
	})
	t.Run("old_profile_and_legacy_SQL_cannot_restore_old_owner", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		var stale model.User
		db.First(&stale, 201)
		op := claimOwnerPair(t, r, "profile-race", "u_201", "u_202")
		if _, err := r.ApplyConfirmed(ctx, op, []string{"u_202"}); err != nil {
			t.Fatal(err)
		}
		stale.Nickname = "edited after migration"
		if err := NewUserRepositoryWithDB(db).Update(&stale); err != nil {
			t.Fatal(err)
		}
		if err := db.Save(&stale).Error; err == nil {
			t.Fatal("old full-row instance bypassed database guard")
		}
		var u model.User
		db.First(&u, 201)
		if u.OwnerID != 110 || u.OwnerRevision != 1 || u.InviteCode != "CODE-A" || u.Nickname != stale.Nickname {
			t.Fatal("profile restored protected tuple")
		}
		if err := NewUserRepositoryWithDB(db).UpdateFields(201, map[string]any{"owner_id": 120}); !errors.Is(err, ErrOwnershipProtected) {
			t.Fatal("public field map bypassed the ownership service")
		}
	})
	t.Run("late_mirror_reloads_creator_and_independent_groups_stay_independent", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		g := model.Group{ID: 301, BizID: 301, IMGroupID: "g_301", OwnerID: 100, CreatedBy: 201, Status: "active"}
		db.Create(&g)
		stale := g
		independent := model.Group{ID: 302, BizID: 302, IMGroupID: "g_302", OwnerID: 120, CreatedBy: 201, OwnerBinding: "independent", Status: "active"}
		db.Create(&independent)
		op := claimOwnerPair(t, r, "mirror-race", "u_201", "u_202")
		if _, err := r.ApplyConfirmed(ctx, op, []string{"u_202"}); err != nil {
			t.Fatal(err)
		}
		stale.Name = "late cloud name"
		if err := NewGroupRepositoryWithDB(db).SaveMirror(ctx, &stale); err != nil {
			t.Fatal(err)
		}
		db.First(&g, 301)
		db.First(&independent, 302)
		if g.OwnerID != 110 || g.OwnerRevision != 1 || independent.OwnerID != 120 {
			t.Fatal("late mirror or migration stole a group")
		}
		late := model.Group{ID: 303, BizID: 303, IMGroupID: "old-instance-late-insert", CreatedBy: 201, OwnerID: 100, Status: "active"}
		if err := db.Create(&late).Error; err == nil {
			t.Fatal("old instance inserted a stale group after owner migration")
		}
		if err := NewGroupRepositoryWithDB(db).SaveMirror(ctx, &late); err != nil {
			t.Fatal(err)
		}
		if late.OwnerID != 110 || late.OwnerRevision != 1 {
			t.Fatal("new mirror did not repair a late creator snapshot")
		}
		g.OwnerID = 120
		g.CreatedBy = 203
		if err := NewGroupRepositoryWithDB(db).Update(&g); err != nil {
			t.Fatal(err)
		}
		var saved model.Group
		db.First(&saved, 301)
		if saved.OwnerID != 110 || saved.CreatedBy != 201 {
			t.Fatal("ordinary group update wrote protected fields")
		}
	})
	t.Run("historical_multi_space_friends_require_review", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		op, err := r.Enqueue(ctx, "reconcile", "history", "u_201", "")
		if err != nil {
			t.Fatal(err)
		}
		op, err = r.Claim(ctx, op.ID)
		if err != nil {
			t.Fatal(err)
		}
		result, err := r.ApplyConfirmed(ctx, op, []string{"u_203", "u_202"})
		if err != nil || result.State != "conflict" || result.ReasonCode != "historical_owner_conflict" {
			t.Fatal("history list order picked an owner")
		}
		var u model.User
		db.First(&u, 201)
		if u.OwnerID != 100 || u.OwnerRevision != 0 {
			t.Fatal("historical conflict changed owner")
		}
	})
	t.Run("invitation_and_admin_transfer_share_versions_and_group_locks", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		db.Create(&model.Group{ID: 301, BizID: 301, IMGroupID: "g_301", OwnerID: 100, CreatedBy: 201, Status: "active"})
		start := make(chan struct{})
		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for _, mode := range []string{"invite", "friend"} {
			wg.Add(1)
			go func(mode string) {
				defer wg.Done()
				<-start
				if mode == "invite" {
					_, err := r.BindInvite(ctx, "u_201", "CODE-B")
					errs <- err
				} else {
					op, err := r.Enqueue(ctx, "client", "invite-race", "u_201", "u_202")
					if err == nil {
						op, err = r.Claim(ctx, op.ID)
					}
					if err == nil {
						_, err = r.ApplyConfirmed(ctx, op, []string{"u_202"})
					}
					errs <- err
				}
			}(mode)
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			var policy OwnershipError
			if err != nil && !errors.As(err, &policy) {
				t.Fatal(err)
			}
		}
		var u model.User
		var g model.Group
		db.First(&u, 201)
		db.First(&g, 301)
		if u.OwnerID != g.OwnerID || u.OwnerRevision != g.OwnerRevision {
			t.Fatal("invitation race split user and group")
		}
		previous := u.OwnerRevision
		result, err := r.Transfer(ctx, 100, 201, 120)
		if err != nil {
			t.Fatal(err)
		}
		db.First(&u, 201)
		db.First(&g, 301)
		if u.OwnerID != 120 || g.OwnerID != 120 || result.Target.OwnerRevision != u.OwnerRevision || u.OwnerRevision < previous {
			t.Fatal("admin transfer lost tuple consistency")
		}
		if _, err := r.Transfer(ctx, 110, 201, 110); err == nil {
			t.Fatal("ordinary manager transferred another manager's user")
		}
	})
	t.Run("same_owner_repairs_only_explicit_follow_groups", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		if err := ownershipTransaction(ctx, db, func(tx *gorm.DB) error {
			return tx.Create(&model.Group{ID: 301, BizID: 301, IMGroupID: "g_301", OwnerID: 100, CreatedBy: 202, OwnerBinding: "follow_user", Status: "active"}).Error
		}); err != nil {
			t.Fatal(err)
		}
		op := claimOwnerPair(t, r, "repair", "u_202", "u_202_other")
		extra := model.User{ID: 204, BizID: 204, Phone: "extra", IMUserID: "u_202_other", Nickname: "same", OwnerID: 110, Status: "active"}
		db.Create(&extra)
		result, err := r.ApplyConfirmed(ctx, op, []string{"u_202_other"})
		if err != nil || result.State != "unchanged" {
			t.Fatal(err)
		}
		var g model.Group
		db.First(&g, 301)
		if g.OwnerID != 110 {
			t.Fatal("legitimate follow group was not repaired")
		}
	})
	t.Run("leases_survive_crash_and_exhaustion_requires_review", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		r := &OwnershipRepository{DB: db}
		old := claimOwnerPair(t, r, "lease", "u_201", "u_202")
		db.Model(old).Update("lease_until", time.Now().Add(-time.Minute))
		if err := r.FinishEffects(ctx, old); !IsOwnershipLeaseLost(err) {
			t.Fatalf("expired lease finished effects: %v", err)
		}
		fresh, err := r.Claim(ctx, old.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ApplyConfirmed(ctx, old, []string{"u_202"}); err == nil {
			t.Fatal("expired worker committed")
		}
		for _, err := range []error{r.FinishEffects(ctx, old), r.Retry(ctx, old, "late_retry"), r.Block(ctx, old, "late_block")} {
			if !IsOwnershipLeaseLost(err) {
				t.Fatalf("old worker changed the re-claimed lease: %v", err)
			}
		}
		if _, err := r.ApplyConfirmed(ctx, fresh, []string{"u_202"}); err != nil {
			t.Fatal(err)
		}
		exhausted := claimOwnerPair(t, r, "exhausted", "u_201", "u_203")
		db.Model(exhausted).Updates(map[string]any{"attempts": 8, "lease_until": time.Now().Add(-time.Minute)})
		if _, err := r.Claim(ctx, exhausted.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("exhausted operation was claimable: %v", err)
		}
		db.First(exhausted, exhausted.ID)
		if exhausted.State != "pending" || exhausted.NeedsReview {
			t.Fatal("request claim ran the maintenance scan")
		}
		if err := r.MarkExpiredForReview(ctx); err != nil {
			t.Fatal(err)
		}
		db.First(exhausted, exhausted.ID)
		if exhausted.State != "manual" || !exhausted.NeedsReview {
			t.Fatal("last-attempt crash was stranded")
		}
	})
	t.Run("schema_reapply_retains_data_and_does_not_duplicate_indexes", func(t *testing.T) {
		db := ownershipTestDB(t)
		ownershipFixtures(t, db)
		count := func() int64 {
			var n int64
			db.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE()").Scan(&n)
			return n
		}
		before := count()
		if err := EnsureOwnershipSchema(db); err != nil {
			t.Fatal(err)
		}
		if err := db.AutoMigrate(&model.User{}, &model.Group{}, &model.FriendOwnerOperation{}); err != nil {
			t.Fatal(err)
		}
		if err := EnsureOwnershipSchema(db); err != nil {
			t.Fatal(err)
		}
		if count() != before {
			t.Fatal("idempotent schema duplicated an index")
		}
		var u model.User
		db.First(&u, 201)
		if u.OwnerRevision != 0 || u.OwnerID != 100 || u.AuthVersion != 7 {
			t.Fatal("schema invented historical assignment or changed auth version")
		}
	})
}
