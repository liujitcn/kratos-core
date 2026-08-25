.PHONY: help init plugin fmt lint test vet api tools wire buf-push tag

GO_BIN_DIR ?= $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
WIRE_DIR ?= .
# normalize-go-imports 对包名 string 的别名规范化会遮蔽 Go 预声明类型，相关文件只交给 goimports。
GO_FILES := $(shell rg --files -g '*.go')
NORMALIZE_GO_FILES := $(filter-out $(shell rg -l '/string"' -g '*.go'),$(GO_FILES))

# 生成 Core 通用 protobuf Go 代码。
api:
	@echo "==> 生成 Core protobuf Go 代码"
	@PATH="$(GO_BIN_DIR):$$PATH" command -v protoc-gen-go >/dev/null 2>&1 || { echo "未找到 protoc-gen-go，请先执行 make tools"; exit 1; }
	@PATH="$(GO_BIN_DIR):$$PATH" command -v protoc-gen-go-errors >/dev/null 2>&1 || { echo "未找到 protoc-gen-go-errors，请先执行 make tools"; exit 1; }
	@PATH="$(GO_BIN_DIR):$$PATH" command -v goimports >/dev/null 2>&1 || { echo "未找到 goimports，请先执行 make tools"; exit 1; }
	@cd api && PATH="$(GO_BIN_DIR):$$PATH" buf generate --template buf.gen.yaml
	@PATH="$(GO_BIN_DIR):$$PATH" find api/gen/go -type f -name '*.go' -exec goimports -w {} +
	@echo "==> Core protobuf Go 代码生成完成"

# 初始化全部开发工具。
init: plugin tools

# 安装并锁定 protobuf 生成插件。
plugin:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v3@v3.0.0-20260626125723-668db92c2c00

# 安装并锁定代码生成与格式化工具。
tools: plugin
	@go install golang.org/x/tools/cmd/goimports@v0.48.0
	@GOBIN="$(GO_BIN_DIR)" go install github.com/liujitcn/kratos-kit/cmd/normalize-go-imports@latest
	@go install github.com/google/wire/cmd/wire@v0.7.0

# 使用 normalize-go-imports 和 goimports 格式化 Go 源码。
fmt:
	@"$(GO_BIN_DIR)/normalize-go-imports" -root . -write $(foreach file,$(NORMALIZE_GO_FILES),-file $(file))
	@echo "==> 格式化 Go 代码"
	@PATH="$(GO_BIN_DIR):$$PATH" goimports -w $$(rg --files -g '*.go')
	@echo "==> Go 代码格式化完成"

# 运行 Core 静态检查。
lint: vet

# 生成 wire 依赖注入代码
wire: fmt
	@if [ ! -f "$(WIRE_DIR)/wire.go" ]; then \
		echo "==> 未找到 $(WIRE_DIR)/wire.go，跳过 wire 依赖注入代码生成"; \
	else \
		echo "==> 生成 wire 依赖注入代码"; \
		PATH="$(GO_BIN_DIR):$$PATH" command -v wire >/dev/null 2>&1 || { echo "未找到 wire，请先执行 make tools"; exit 1; }; \
		cd "$(WIRE_DIR)" && PATH="$(GO_BIN_DIR):$$PATH" wire . && \
		PATH="$(GO_BIN_DIR):$$PATH" goimports -w wire.go wire_gen.go && \
		echo "==> wire 依赖注入代码生成完成"; \
	fi

# 验证核心模块。
test:
	@go test ./...
	@cd api && go test ./...
	@cd client && go test ./...

# 静态检查核心模块。
vet:
	@go vet ./...
	@cd api && go vet ./...
	@cd client && go vet ./...

# 推送 Core protobuf 模块到 Buf Schema Registry。
buf-push: api fmt
	@if ! buf registry whoami buf.build >/dev/null 2>&1; then \
		echo "未登录 Buf Schema Registry，本次暂不推送，请先执行："; \
		echo "  buf registry login buf.build --token-stdin"; \
		exit 0; \
	fi; \
	$(MAKE) fmt && cd api && buf push --label main

# 统一打 tag：默认递归扫描；MODULE 指定起始目录，EXACT=1 时仅处理该 module（不提交代码）
tag:
	@python3 scripts/tag_release.py $(if $(MODULE),--path $(MODULE),) $(if $(EXACT),--exact,)

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
