package controller_chat

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	mysqlconfig "github.com/go-sql-driver/mysql"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ownerProofFixture struct {
	check func(context.Context, string, []string) (map[string]bool, error)
}

func (p ownerProofFixture) CheckBilateral(c context.Context, u string, ids []string) (map[string]bool, error) {
	return p.check(c, u, ids)
}
func (p ownerProofFixture) ListFriendIDsContext(context.Context, string) ([]string, error) {
	return []string{"peer"}, nil
}

func serviceOwnerDB(t *testing.T) *gorm.DB {
	t.Helper()
	raw := os.Getenv("RUI_OWNER_MYSQL_DSN")
	if raw == "" {
		t.Skip("requires disposable loopback MySQL")
	}
	cfg, err := mysqlconfig.ParseDSN(raw)
	if err != nil || cfg.Net != "tcp" || !strings.HasPrefix(cfg.Addr, "127.0.0.1:") || !strings.HasPrefix(cfg.DBName, "rui_owner_test") {
		t.Fatal("refusing non-isolated database")
	}
	cfg.DBName = ""
	admin, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated MySQL unavailable")
	}
	name := "rui_owner_test_" + repository.OwnershipID()[:12]
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatal(err)
	}
	cfg.DBName = name
	db, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated schema unavailable")
	}
	t.Cleanup(func() {
		c, _ := db.DB()
		_ = c.Close()
		_ = admin.Exec("DROP DATABASE " + name).Error
		a, _ := admin.DB()
		_ = a.Close()
	})
	if err := db.AutoMigrate(&model.User{}, &model.AdminUser{}, &model.InviteCode{}, &model.Group{}, &model.FriendOwnerOperation{}); err != nil {
		t.Fatal(err)
	}
	if err := repository.EnsureOwnershipSchema(db); err != nil {
		t.Fatal(err)
	}
	for _, a := range []model.AdminUser{{ID: 101, Username: "A", Role: "admin", Status: "active"}} {
		if err := db.Create(&a).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []model.User{{ID: 201, BizID: 201, Phone: "pool", IMUserID: "pool", Nickname: "pool", Status: "active"}, {ID: 202, BizID: 202, Phone: "peer", IMUserID: "peer", Nickname: "peer", Status: "active", OwnerID: 101}} {
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestOwnerServiceMySQL(t *testing.T) {
	t.Run("database_failure_is_durable_then_recovers", func(t *testing.T) {
		db := serviceOwnerDB(t)
		ctx := context.Background()
		r := &repository.OwnershipRepository{DB: db}
		row, _ := r.Enqueue(ctx, "client", "db-failure", "pool", "peer")
		op, err := r.Claim(ctx, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec("CREATE TRIGGER owner_service_failure BEFORE UPDATE ON users FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='injected_failure'").Error; err != nil {
			t.Fatal(err)
		}
		proof := ownerProofFixture{check: func(context.Context, string, []string) (map[string]bool, error) {
			return map[string]bool{"peer": true}, nil
		}}
		service := OwnerService{Repo: r, Proof: proof}
		if err := service.Process(ctx, op); err != nil {
			t.Fatal(err)
		}
		var user model.User
		db.First(&user, 201)
		db.First(row, row.ID)
		if user.OwnerID != 0 || row.State != "pending" || row.ReasonCode != "assignment_database_unavailable" || row.LeaseToken != "" {
			t.Fatal("DB failure was not durably pending")
		}
		if err := db.Exec("DROP TRIGGER owner_service_failure").Error; err != nil {
			t.Fatal(err)
		}
		db.Model(row).Update("available_at", time.Now().Add(-time.Second))
		fresh := &repository.OwnershipRepository{DB: db}
		op, err = fresh.Claim(ctx, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		service = OwnerService{Repo: fresh, Proof: proof}
		if err := service.Process(ctx, op); err != nil {
			t.Fatal(err)
		}
		db.First(&user, 201)
		if user.OwnerID != 101 || user.OwnerRevision != 1 {
			t.Fatal("new service did not recover the durable DB failure")
		}
	})
	t.Run("pending_proof_never_assigns_then_new_worker_recovers", func(t *testing.T) {
		db := serviceOwnerDB(t)
		ctx := context.Background()
		r := &repository.OwnershipRepository{DB: db}
		row, err := r.Enqueue(ctx, "client", "pending", "pool", "peer")
		if err != nil {
			t.Fatal(err)
		}
		op, err := r.Claim(ctx, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		effects := 0
		service := OwnerService{Repo: r, Proof: ownerProofFixture{check: func(context.Context, string, []string) (map[string]bool, error) {
			return map[string]bool{"peer": false}, nil
		}}, Effects: func(context.Context, *model.OwnerAssignment) error { effects++; return nil }}
		if err := service.Process(ctx, op); err != nil {
			t.Fatal(err)
		}
		var user model.User
		db.First(&user, 201)
		db.First(row, row.ID)
		if user.OwnerID != 0 || user.OwnerRevision != 0 || row.State != "pending" || row.LeaseToken != "" || effects != 0 {
			t.Fatal("unconfirmed request changed assignment or lost durable retry")
		}
		db.Model(row).Update("available_at", time.Now().Add(-time.Second))
		r = &repository.OwnershipRepository{DB: db}
		op, err = r.Claim(ctx, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		service.Repo = r
		service.Proof = ownerProofFixture{check: func(ctx context.Context, _ string, _ []string) (map[string]bool, error) {
			// A second connection must acquire this user while the remote proof runs.
			probe, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			err := db.WithContext(probe).Transaction(func(tx *gorm.DB) error { return tx.Exec("SELECT id FROM users WHERE id=201 FOR UPDATE NOWAIT").Error })
			if err != nil {
				return nil, errors.New("RPC held a business row lock")
			}
			return map[string]bool{"peer": true}, nil
		}}
		if err := service.Process(ctx, op); err != nil {
			t.Fatal(err)
		}
		db.First(&user, 201)
		db.First(row, row.ID)
		if user.OwnerID != 101 || user.OwnerRevision != 1 || row.State != "migrated" || row.EffectsPending || effects != 1 {
			t.Fatal("new worker did not recover a confirmed relationship")
		}
	})
	t.Run("committed_assignment_survives_effect_failure", func(t *testing.T) {
		db := serviceOwnerDB(t)
		ctx := context.Background()
		r := &repository.OwnershipRepository{DB: db}
		row, _ := r.Enqueue(ctx, "client", "effects", "pool", "peer")
		op, err := r.Claim(ctx, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		proofCalls := 0
		service := OwnerService{Repo: r, Proof: ownerProofFixture{check: func(context.Context, string, []string) (map[string]bool, error) {
			proofCalls++
			return map[string]bool{"peer": true}, nil
		}}, Effects: func(context.Context, *model.OwnerAssignment) error { return errors.New("injected effect failure") }}
		if err := service.Process(ctx, op); err != nil {
			t.Fatal(err)
		}
		var u model.User
		db.First(&u, 201)
		db.First(row, row.ID)
		if u.OwnerRevision != 1 || row.State != "migrated" || !row.EffectsPending {
			t.Fatal("post-commit error undid friendship assignment")
		}
		db.Model(row).Update("available_at", time.Now().Add(-time.Second))
		op, err = r.Claim(ctx, row.ID)
		if err != nil {
			t.Fatal(err)
		}
		service.Effects = func(context.Context, *model.OwnerAssignment) error { return nil }
		if err := service.Process(ctx, op); err != nil {
			t.Fatal(err)
		}
		db.First(&u, 201)
		db.First(row, row.ID)
		if u.OwnerRevision != 1 || row.EffectsPending || proofCalls != 1 {
			t.Fatal("effect retry repeated migration or proof")
		}
	})
}

func TestOwnerServiceDeadlockKeepsProofAndEffectsOutsideDatabaseRetryMySQL(t *testing.T) {
	for _, test := range []struct {
		name  string
		stage string
		fails int
	}{
		{name: "apply rolls back and retries", stage: "apply", fails: 1},
		{name: "finish retries without replaying effects", stage: "finish", fails: 1},
		{name: "apply exhaustion returns failure", stage: "apply", fails: 3},
		{name: "finish exhaustion leaves committed work recoverable", stage: "finish", fails: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := serviceOwnerDB(t)
			ctx := context.Background()
			repo := &repository.OwnershipRepository{DB: db}
			queued, err := repo.Enqueue(ctx, "client", "deadlock", "pool", "peer")
			if err != nil {
				t.Fatal(err)
			}
			job, err := repo.Claim(ctx, queued.ID)
			if err != nil {
				t.Fatal(err)
			}
			remaining := test.fails
			if err := db.Callback().Update().Before("gorm:update").Register("test:inject_ownership_1213", func(tx *gorm.DB) {
				if tx.Statement.Table != "friend_owner_operations" || remaining == 0 {
					return
				}
				fields, _ := tx.Statement.Dest.(map[string]any)
				_, applying := fields["result_json"]
				finishing := fields["effects_pending"] == false && fields["last_error"] == ""
				if (test.stage == "apply" && applying) || (test.stage == "finish" && finishing) {
					remaining--
					_ = tx.AddError(&mysqlconfig.MySQLError{Number: 1213})
				}
			}); err != nil {
				t.Fatal(err)
			}

			proofCalls, effectCalls := 0, 0
			service := OwnerService{
				Repo: repo,
				Proof: ownerProofFixture{check: func(context.Context, string, []string) (map[string]bool, error) {
					proofCalls++
					return map[string]bool{"peer": true}, nil
				}},
				Effects: func(context.Context, *model.OwnerAssignment) error { effectCalls++; return nil },
			}
			err = service.Process(ctx, job)
			exhausted := test.fails == 3
			if remaining != 0 || proofCalls != 1 || (exhausted && !repository.IsOwnershipDeadlock(err)) || (!exhausted && err != nil) {
				t.Fatalf("remaining=%d proof calls=%d err=%v", remaining, proofCalls, err)
			}
			var row model.FriendOwnerOperation
			if err := db.First(&row, job.ID).Error; err != nil {
				t.Fatal(err)
			}
			var user model.User
			if err := db.First(&user, 201).Error; err != nil {
				t.Fatal(err)
			}
			if row.Attempts != 1 {
				t.Fatalf("database retries consumed business attempts: %d", row.Attempts)
			}
			if exhausted && test.stage == "apply" {
				if row.State != "pending" || row.LeaseToken != "" || effectCalls != 0 || user.OwnerRevision != 0 {
					t.Fatal("exhausted apply did not roll back and durably reschedule")
				}
				return
			}
			if row.State != "migrated" || effectCalls != 1 || user.OwnerRevision != 1 || row.EffectsPending != exhausted {
				t.Fatalf("unexpected commit or repeated effects: state=%s effects=%d revision=%d pending=%v", row.State, effectCalls, user.OwnerRevision, row.EffectsPending)
			}
			if exhausted {
				if err := db.Model(&row).Update("lease_until", time.Now().Add(-time.Minute)).Error; err != nil {
					t.Fatal(err)
				}
				// A restarted worker resumes the outbox. The real effect handlers
				// invalidate caches and upsert state, so replay after a crash is safe.
				fresh := &repository.OwnershipRepository{DB: db}
				service.Repo = fresh
				job, err = fresh.Claim(ctx, row.ID)
				if err != nil {
					t.Fatal(err)
				}
				if err := service.Process(ctx, job); err != nil {
					t.Fatal(err)
				}
				if err := db.First(&row, row.ID).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.First(&user, 201).Error; err != nil {
					t.Fatal(err)
				}
				if proofCalls != 1 || user.OwnerRevision != 1 || row.EffectsPending || row.LeaseToken != "" {
					t.Fatal("outbox recovery repeated proof or migration, or left the job stuck")
				}
			}
		})
	}
}
