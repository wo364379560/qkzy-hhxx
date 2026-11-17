package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	controller_chat "github.com/ixayda/ix-Go/internal/app/server/controller/chat"
	"github.com/ixayda/ix-Go/internal/app/server/forwardqueue"
	handler_chat "github.com/ixayda/ix-Go/internal/app/server/handler/chat"
	"github.com/ixayda/ix-Go/internal/app/server/presence"
	"github.com/ixayda/ix-Go/internal/app/server/router"
	"github.com/ixayda/ix-Go/internal/pkg/app/setup"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/internal/pkg/datasource"
	"github.com/ixayda/ix-Go/pkg/logger"
)

// Provider 实现 app.ServiceProvider 接口。
type Provider struct{}

func (Provider) Usage() string {
	var buffer strings.Builder
	return buffer.String()
}

func (Provider) Flags() (bool, error) {
	return false, nil
}

func (Provider) Execute() error {
	cfg := config.Cfg.Get().App
	readOnlyStandby := readOnlyStandbyEnabled()

	datasource.Datasource.Init()
	if readOnlyStandby {
		logger.Warnf("read-only standby mode enabled: skipping startup database writes and presence worker")
	} else {
		setup.Database()
		controller_chat.EnsureSecurityAssistantAccount()
	}

	var stopWorkers context.CancelFunc
	if !readOnlyStandby {
		workerCtx, cancelWorkers := context.WithCancel(context.Background())
		stopWorkers = cancelWorkers
		defer stopWorkers()
		presence.StartLastOnlineWorker(workerCtx)
		handler_chat.StartAttachmentMultipartWorker(workerCtx, datasource.Datasource.Mysql.GetClient())
		forwardqueue.Start(workerCtx, datasource.Datasource.Mysql.GetClient())
		controller_chat.StartFriendOwnerWorker(workerCtx, datasource.Datasource.Mysql.GetClient())
	}

	r := router.New()

	addr := net.JoinHostPort(cfg.Addr, cfg.Port)
	listener, err := net.Listen(detectListenNetwork(cfg.Addr), addr)
	if err != nil {
		return fmt.Errorf("listen %s failed: %w", addr, err)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		// 后台 CSV 导出和批量腾讯 IM 操作可能合法超过 30 秒；公开认证
		// 接口另有 Nginx/Go 限频和并发保险丝，不依赖全局写超时抗 CC。
		WriteTimeout:   120 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 16 << 10,
	}

	logger.Infof("Server Listening %s ...", listener.Addr().String())

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Panicf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Infof("Server shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(ctx)
	if !readOnlyStandby {
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
		presence.FlushLastOnline(flushCtx)
		flushCancel()
		stopWorkers()
	}
	return shutdownErr
}

func readOnlyStandbyEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("CHENGNAN_STANDBY_READ_ONLY"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func detectListenNetwork(host string) string {
	ip := net.ParseIP(host)
	if ip == nil {
		return "tcp"
	}
	if ip.To4() != nil {
		return "tcp4"
	}
	return "tcp6"
}
