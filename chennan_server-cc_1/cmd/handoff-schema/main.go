// Generates reviewable additive SQL from the actual Rui models in an isolated
// database. It never loads .env or accepts a production database address.
package main

import (
	"fmt"
	driver "github.com/go-sql-driver/mysql"
	"github.com/ixayda/ix-Go/internal/app/server/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"regexp"
	"strings"
)

func main() {
	dsn := os.Getenv("RUI_HANDOFF_MIGRATION_DSN")
	cfg, err := driver.ParseDSN(dsn)
	if err != nil || cfg == nil || cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:33326" || cfg.DBName != "rui_handoff_review" {
		panic("only isolated 127.0.0.1:33326/rui_handoff_review is allowed")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	must(err)
	must(db.AutoMigrate(&model.ForwardJob{}, &model.ForwardDelivery{}, &model.ForwardTarget{}, &model.ForwardMessage{}, &model.ForwardRetry{}, &model.ForwardGate{}, &model.ChatMultipartUpload{}, &model.UserRefreshSession{}, &model.GroupNotice{}))
	var output strings.Builder
	output.WriteString("-- Rui additive reliability migration, 2026-11-01.\n-- Reviewed against MySQL 8.0 in an isolated database. Back up production first.\n-- Existing chat_attachments/users tables stay in place. Do not run both this file and the optional recovery ALTER script.\n\n")
	for _, table := range []string{"forward_jobs", "forward_targets", "forward_messages", "forward_deliveries", "forward_retries", "forward_gates", "chat_multipart_uploads"} {
		var name, ddl string
		must(db.Raw("SHOW CREATE TABLE `"+table+"`").Row().Scan(&name, &ddl))
		ddl = strings.Replace(ddl, "CREATE TABLE ", "CREATE TABLE IF NOT EXISTS ", 1)
		ddl = regexp.MustCompile(` AUTO_INCREMENT=\d+`).ReplaceAllString(ddl, "")
		output.WriteString(ddl + ";\n\n")
	}
	for _, column := range []struct{ table, name, definition string }{
		{"user_refresh_sessions", "previous_refresh_token_hash", "varchar(64) NULL"},
		{"user_refresh_sessions", "previous_refresh_expires_at", "datetime(3) NULL"},
		{"user_refresh_sessions", "refresh_token_ciphertext", "varchar(512) NULL"},
		{"group_notices", "pinned", "tinyint(1) NOT NULL DEFAULT 0"},
	} {
		ddl := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", column.table, column.name, column.definition)
		output.WriteString(fmt.Sprintf("SET @rui_ddl = (SELECT IF(COUNT(*) = 0, '%s', 'SELECT 1') FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s' AND COLUMN_NAME = '%s');\nPREPARE rui_stmt FROM @rui_ddl;\nEXECUTE rui_stmt;\nDEALLOCATE PREPARE rui_stmt;\n\n", ddl, column.table, column.name))
	}
	output.WriteString("SET @rui_ddl = (SELECT IF(COUNT(*) = 0, 'CREATE INDEX idx_user_refresh_sessions_previous_refresh_token_hash ON user_refresh_sessions(previous_refresh_token_hash)', 'SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_refresh_sessions' AND INDEX_NAME = 'idx_user_refresh_sessions_previous_refresh_token_hash');\nPREPARE rui_stmt FROM @rui_ddl;\nEXECUTE rui_stmt;\nDEALLOCATE PREPARE rui_stmt;\n")
	fmt.Print(output.String())
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
