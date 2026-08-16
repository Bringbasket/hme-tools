# GoKeep 统一入口（Windows 下请用 pnpm 脚本或手动执行对应命令）
# 详见 docs/ 与 README.md

GO  := go
GOBIN ?= $(shell go env GOPATH)/bin

.PHONY: help dev web-dev web-build web-typecheck gen-ent server-run server-build lint test ci

help:
	@echo "make dev           启动前端 dev server（需同时启动 server）"
	@echo "make web-dev       Vite dev（代理 /api 到 :8080）"
	@echo "make web-build     构建前端到 server/webui/dist"
	@echo "make web-typecheck vue-tsc 类型检查"
	@echo "make gen-ent       生成 Ent 代码（schema 变更后必须执行）"
	@echo "make server-run    运行 Go 网关（默认 :8080）"
	@echo "make server-build  编译 Go 二进制"
	@echo "make lint          golangci-lint（可选安装）+ pnpm lint"
	@echo "make ci            本地跑一遍 CI 检查"

dev:
	pnpm --filter @gokeep/web dev

web-build:
	pnpm --filter @gokeep/web build

web-typecheck:
	pnpm --filter @gokeep/web typecheck

gen-ent:
	cd server && go generate ./internal/ent

server-run:
	cd server && go run ./cmd/server

server-build:
	cd server && go build -o bin/gokeep-server ./cmd/server

lint:
	cd server && go vet ./...
	pnpm lint

ci:
	pnpm --filter @gokeep/web typecheck
	pnpm --filter @gokeep/web build
	cd server && go vet ./... && go test ./...
