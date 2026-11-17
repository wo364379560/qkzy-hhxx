// ownership-migrate defaults to inspection. -apply performs additive ownership
// migration only; it never seeds accounts or calls Tencent.
package main

import (
	"flag"
	"fmt"
	"os"

	mysqlconfig "github.com/go-sql-driver/mysql"
	"github.com/ixayda/ix-Go/internal/app/server/repository"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	apply := flag.Bool("apply", false, "apply additive ownership migration (back up database first)")
	flag.Parse()
	if err := config.Setup(); err != nil {
		fail("configuration invalid")
	}
	opts := config.Cfg.Get().Mysql
	cfg := mysqlconfig.NewConfig()
	cfg.User = opts.User
	cfg.Passwd = opts.Passwd
	cfg.Net = "tcp"
	cfg.Addr = opts.Addr + ":" + opts.Port
	cfg.DBName = opts.Database
	cfg.ParseTime = true
	db, err := gorm.Open(mysqlgorm.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fail("database unavailable (connection details redacted)")
	}
	if !db.Migrator().HasTable("users") || !db.Migrator().HasTable("groups") || !db.Migrator().HasTable("invite_codes") {
		fail("expected existing business tables; refusing empty/wrong database")
	}
	if *apply {
		if err := repository.MigrateOwnershipSchema(db); err != nil {
			fail("ownership migration failed; inspect database grants and schema; no historical owner backfill was performed")
		}
		fmt.Println("ownership migration applied; historical owner/invitation/authentication data retained")
	} else {
		for _, entry := range []struct{ table, column string }{{"users", "owner_revision"}, {"groups", "owner_binding"}, {"groups", "owner_revision"}, {"groups", "mirror_revision"}, {"friend_owner_operations", "operation_id"}} {
			fmt.Printf("%s.%s present=%v\n", entry.table, entry.column, db.Migrator().HasColumn(entry.table, entry.column))
		}
		fmt.Println("inspection only; use -apply after backup and reviewing the rollout instructions")
	}
}
func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
