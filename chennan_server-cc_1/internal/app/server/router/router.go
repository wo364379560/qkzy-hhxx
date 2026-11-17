package router

import (
	"github.com/gin-gonic/gin"
	handler_public "github.com/ixayda/ix-Go/internal/app/server/handler/public"
	"github.com/ixayda/ix-Go/internal/app/server/handler/webhook"
	"github.com/ixayda/ix-Go/internal/app/server/middleware"
	"github.com/ixayda/ix-Go/internal/pkg/handlers"
)

// New 初始化 Gin engine 并注册所有路由。
func New() *gin.Engine {
	r := gin.New()
	handlers.SetHandlers(r)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.APINoStore())
	v1.Use(middleware.SourceMigrationFence())
	v1.Use(middleware.PublicProtection())
	v1.Use(middleware.IPAccessBlock())
	RegisterChat(v1)
	RegisterAdmin(v1)
	v1.GET("/public/bootstrap/domain-pool", handler_public.GetBackendDomainPoolBootstrap)

	v1.POST("/webhook/tencent", handler_webhook.TencentCallback)

	return r
}
