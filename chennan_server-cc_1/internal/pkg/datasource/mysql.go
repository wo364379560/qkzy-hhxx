package datasource

import (
	"database/sql"
	"fmt"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/handlers"
	"github.com/ixayda/ix-Go/pkg/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type mysqlClient struct {
	client *gorm.DB
}

func (m *mysqlClient) GetClient() *gorm.DB {
	return m.client
}

func (m *mysqlClient) openMysql() {
	options := config.Cfg.Get()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s",
		options.Mysql.User,
		options.Mysql.Passwd,
		options.Mysql.Addr,
		options.Mysql.Port,
		options.Mysql.Database,
		options.Mysql.Options,
	)

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Panicf("failed to open mysql connection: %v", err)
	}

	sqlDB.SetMaxOpenConns(options.Mysql.MaxOpenConns)
	sqlDB.SetMaxIdleConns(options.Mysql.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(options.Mysql.ConnMaxIdleTime)
	sqlDB.SetConnMaxLifetime(options.Mysql.ConnMaxLifetime)

	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), &gorm.Config{
		Logger: handlers.GormLogger(),
	})

	if err != nil {
		logger.Panicf("failed to initialize database, got error %v", err)
	}

	logger.Infof("Mysql Use %s:%s [%s]\n", options.Mysql.Addr, options.Mysql.Port, "OK")

	m.client = db
}
