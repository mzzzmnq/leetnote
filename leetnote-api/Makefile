APP     := leetnote-api
CMD     := ./cmd/server
BIN     := bin/$(APP)
MODULE  := github.com/mzzzmnq/leetnote-api

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X $(MODULE)/internal/version.Version=$(VERSION) \
           -X $(MODULE)/internal/version.Commit=$(COMMIT) \
           -X $(MODULE)/internal/version.BuildTime=$(DATE)

.PHONY: help
help: ## 显示所有可用命令
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## 本地运行
	go run $(CMD)

.PHONY: build
build: ## 编译到 bin/
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD)

.PHONY: test
test: ## 跑单元测试
	go test ./... -race -cover

.PHONY: cover
cover: ## 生成 HTML 覆盖率报告
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: lint
lint: ## 静态检查（需先装 golangci-lint）
	golangci-lint run ./...

.PHONY: fmt
fmt: ## 格式化 + 整理依赖
	go fmt ./...
	go mod tidy

.PHONY: tidy
tidy: ## 整理依赖
	go mod tidy

.PHONY: migrate-up
migrate-up: ## 应用全部迁移（需先装 golang-migrate）
	migrate -path migrations -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## 回滚一个迁移版本
	migrate -path migrations -database "$(DATABASE_URL)" down 1

.PHONY: clean
clean: ## 清理编译产物
	rm -rf bin/ coverage.out coverage.html
