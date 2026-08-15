.PHONY: help fmt lint test vet api wire buf-push

GO_BIN_DIR ?= $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
WIRE_DIR ?= .

# 生成 Core 通用 protobuf Go 代码。
api:
	@echo "==> 生成 Core protobuf Go 代码"
	@cd api && PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.gen.yaml
	@PATH="$(GO_BIN_DIR):$$PATH" find api/gen/go -type f -name '*.go' -exec goimports -w {} +
	@echo "==> Core protobuf Go 代码生成完成"

# 格式化 Go 源码。
fmt:
	@PATH="$(GO_BIN_DIR):$$PATH" find . -type f -name '*.go' -exec goimports -w {} +

# 运行 Core 静态检查。
lint: vet

# 生成 wire 依赖注入代码
wire:
	@if [ ! -f "$(WIRE_DIR)/wire.go" ]; then \
		echo "==> 未找到 $(WIRE_DIR)/wire.go，跳过 wire 依赖注入代码生成"; \
	else \
		echo "==> 生成 wire 依赖注入代码"; \
		cd "$(WIRE_DIR)" && PATH="$(GO_BIN_DIR):$$PATH" wire . && \
		PATH="$(GO_BIN_DIR):$$PATH" goimports -w wire.go wire_gen.go && \
		echo "==> wire 依赖注入代码生成完成"; \
	fi

# 验证核心模块。
test:
	@go test ./...

# 静态检查核心模块。
vet:
	@go vet ./...

# 推送 Core protobuf 模块到 Buf Schema Registry。
buf-push: api fmt
	@if ! buf registry whoami buf.build >/dev/null 2>&1; then \
		echo "未登录 Buf Schema Registry，本次暂不推送，请先执行："; \
		echo "  buf registry login buf.build --token-stdin"; \
		exit 0; \
	fi; \
	$(MAKE) fmt && cd api && buf push --label main

# 显示帮助
help:
	@echo ""
	@echo "Usage:"
	@echo " make [target]"
	@echo ""
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
