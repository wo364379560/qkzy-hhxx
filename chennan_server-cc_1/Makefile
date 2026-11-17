PREFIX := txim
DOCKER_GO_IMAGE ?= golang:1.26

# 自动发现：cmd/*/main.go → Go 微服务名（如 server）
GO_SERVICES   := $(patsubst cmd/%/main.go, %, $(wildcard cmd/*/main.go))
ALL_SERVICES  := $(GO_SERVICES)

.PHONY: all $(ALL_SERVICES) server-linux server-linux-arm64 run test clean help

all: $(ALL_SERVICES)        ## 构建全部 Go 服务（本机架构，勿直接丢 Linux）

$(GO_SERVICES):             ## 构建单个 Go 服务，如 make server（本机架构）
	@./scripts/build.sh $(PREFIX) $@

# Docker 里交叉编译。默认只打 amd64（宝塔 CentOS x86）；ARM 需显式 make server-linux-arm64
define docker-build
	docker run --rm \
		-v "$(CURDIR)":/src \
		-v txim-go-mod:/go/pkg/mod \
		-w /src \
		-e GOPROXY=https://goproxy.cn,direct \
		-e GOSUMDB=sum.golang.google.cn \
		$(DOCKER_GO_IMAGE) \
		sh -c 'GOOS=linux GOARCH=$(1) ./scripts/build.sh $(PREFIX) server'
endef

server-linux:               ## Docker 打 Linux amd64（宝塔 CentOS x86）
	$(call docker-build,amd64)
	@echo ""
	@echo "宝塔 CentOS x86 用:  bin/txim-server-linux-amd64"

server-linux-arm64:         ## Docker 打 Linux arm64（仅明确要求时）
	$(call docker-build,arm64)
	@echo ""
	@echo "Linux ARM64 用:     bin/txim-server-linux-arm64"

run:                        ## 开发模式运行 server
	@go run cmd/server/main.go

test:                       ## 运行 Go 测试
	@go test ./...

clean:                      ## 清理构建产物
	@rm -rf bin
	@echo "clean [ ok ]"

help:                       ## 显示帮助
	@echo "Services: $(ALL_SERVICES)\n"
	@echo "Deploy:   make server-linux        # Docker 打 Linux amd64（宝塔默认）\n"
	@echo "          make server-linux-arm64  # 仅在需要 ARM 时\n"
	@echo "          宝塔 CentOS x86 上传 bin/txim-server-linux-amd64\n"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
