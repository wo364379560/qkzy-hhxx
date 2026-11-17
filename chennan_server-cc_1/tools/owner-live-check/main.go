// This verification server can only write to a disposable loopback database.
// It uses real Tencent bilateral checks for explicitly supplied test accounts.
// No account import, profile change, friend add or production DDL is performed.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	mysqlconfig "github.com/go-sql-driver/mysql"
	chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/app/server/tencentim"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fixture struct {
	Alias string `json:"alias"`
	Auth  struct {
		UserID string `json:"user_id"`
	} `json:"auth"`
}

func main() {
	cfg, err := mysqlconfig.ParseDSN(os.Getenv("RUI_OWNER_MYSQL_DSN"))
	if err != nil || cfg.Net != "tcp" || !strings.HasPrefix(cfg.Addr, "127.0.0.1:") || !strings.HasPrefix(cfg.DBName, "rui_owner_test") {
		panic("refusing non-isolated database")
	}
	if err := config.Setup(); err != nil {
		panic("invalid configuration")
	}
	db, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		panic("isolated MySQL unavailable")
	}
	// Every run gets its own schema. Existing runs and their evidence are retained.
	name := "rui_owner_test_live_" + repository.OwnershipID()[:12]
	if err := db.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4").Error; err != nil {
		panic("cannot create isolated schema")
	}
	base, _ := db.DB()
	_ = base.Close()
	cfg.DBName = name
	db, err = gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		panic("isolated verification schema unavailable")
	}
	raw, err := os.ReadFile(os.Getenv("RUI_OWNER_TEST_ACCOUNTS"))
	if err != nil {
		panic("private account fixture missing")
	}
	var accounts []fixture
	if json.Unmarshal(raw, &accounts) != nil || len(accounts) != 5 {
		panic("expected five explicitly supplied test accounts")
	}
	if err := db.AutoMigrate(&model.AdminUser{}, &model.User{}, &model.InviteCode{}, &model.Group{}, &model.FriendOwnerOperation{}); err != nil {
		panic("schema setup failed")
	}
	if err := repository.EnsureOwnershipSchema(db); err != nil {
		panic("ownership guards failed")
	}
	for _, a := range []model.AdminUser{{ID: 100, Username: "fixture-super", Role: "super_admin", Status: "active"}, {ID: 110, Username: "fixture-a", Role: "admin", Status: "active"}, {ID: 120, Username: "fixture-b", Role: "admin", Status: "active"}} {
		must(db.Create(&a).Error)
	}
	users := map[string]model.User{}
	for i, a := range accounts {
		if a.Alias != fmt.Sprintf("T%d", i+1) || a.Auth.UserID == "" {
			panic("invalid fixture mapping")
		}
		owner := uint(0)
		if i == 1 {
			owner = 110
		}
		if i == 2 {
			owner = 120
		}
		u := model.User{ID: uint(201 + i), BizID: int64(201 + i), Phone: "isolated-" + a.Alias, IMUserID: a.Auth.UserID, Nickname: a.Alias, Status: "active", OwnerID: owner, AuthVersion: 7, GloballySearchable: true}
		must(db.Create(&u).Error)
		users[a.Alias] = u
	}
	for _, code := range []model.InviteCode{{Code: "FIXTURE-A", OwnerID: 110, Status: "active"}, {Code: "FIXTURE-B", OwnerID: 120, Status: "active"}} {
		must(db.Create(&code).Error)
	}
	must(db.Create(&model.Group{ID: 301, BizID: 301, IMGroupID: "isolated-follow-T1", CreatedBy: 201, OwnerID: 0, OwnerBinding: "follow_user", Status: "active"}).Error)
	must(db.Create(&model.Group{ID: 302, BizID: 302, IMGroupID: "isolated-independent-T1", CreatedBy: 201, OwnerID: 120, OwnerBinding: "independent", Status: "active"}).Error)
	repo := &repository.OwnershipRepository{DB: db}
	service := chat.OwnerService{Repo: repo, Proof: tencentim.NewClient()}
	token := repository.OwnershipID()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(403)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var input struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Request string `json:"request"`
			Visible bool   `json:"visible"`
		}
		if r.Method == "POST" {
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil {
				w.WriteHeader(400)
				return
			}
		}
		from, ok := users[input.From]
		target, ok2 := users[input.To]
		output := map[string]any{}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		switch r.URL.Path {
		case "/state":
			list := []map[string]any{}
			for i := 0; i < 5; i++ {
				alias := fmt.Sprintf("T%d", i+1)
				u := users[alias]
				must(db.First(&u, u.ID).Error)
				list = append(list, map[string]any{"alias": alias, "owner_id": u.OwnerID, "revision": u.OwnerRevision, "invite_code": u.InviteCode, "auth_version": u.AuthVersion})
			}
			var groups []model.Group
			db.Order("id").Find(&groups)
			output["users"] = list
			for i := range groups {
				groups[i].IMGroupID = ""
			}
			output["groups"] = groups
		case "/visible":
			if !ok {
				w.WriteHeader(400)
				return
			}
			must(db.Model(&model.User{}).Where("id=?", from.ID).Update("globally_searchable", input.Visible).Error)
			output["updated"] = true
		case "/policy":
			if !ok || !ok2 {
				w.WriteHeader(400)
				return
			}
			db.First(&from, from.ID)
			db.First(&target, target.ID)
			var a, b model.AdminUser
			db.First(&a, from.OwnerID)
			db.First(&b, target.OwnerID)
			output["reason"] = model.DiscoverOwnerPair(&from, &target, model.ClassifyOwner(&from, &a), model.ClassifyOwner(&target, &b))
		case "/sync":
			if !ok || !ok2 {
				w.WriteHeader(400)
				return
			}
			op, err := repo.Enqueue(ctx, "client", input.Request, from.IMUserID, target.IMUserID)
			if err == nil {
				var claimed *model.FriendOwnerOperation
				claimed, err = repo.Claim(ctx, op.ID)
				if err == nil {
					err = service.Process(ctx, claimed)
				} else if err == gorm.ErrRecordNotFound {
					err = nil
				}
			}
			if err != nil {
				w.WriteHeader(503)
				json.NewEncoder(w).Encode(map[string]string{"error": "verification_unavailable"})
				return
			}
			result, err := repo.Assignment(ctx, op.ID)
			if err != nil {
				w.WriteHeader(503)
				return
			}
			output["state"] = result.State
			output["reason"] = result.ReasonCode
			output["operation_id"] = result.RequestID
			output["groups"] = result.AffectedGroups
		default:
			w.WriteHeader(404)
			return
		}
		json.NewEncoder(w).Encode(output)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("cannot listen")
	}
	endpoint, _ := json.Marshal(map[string]string{"url": "http://" + listener.Addr().String(), "token": token})
	must(os.WriteFile(os.Getenv("RUI_OWNER_VERIFY_ENDPOINT"), endpoint, 0600))
	fmt.Println("isolated ownership verification ready")
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	must(server.Serve(listener))
}
func must(err error) {
	if err != nil {
		panic("isolated verification operation failed")
	}
}
