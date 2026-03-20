GOHOSTOS := $(shell go env GOHOSTOS)
GOPATH := $(shell go env GOPATH)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || printf 'v0.9.9')
GIT_VERSION := $(shell git describe --tags 2>/dev/null || printf 'dev')
PREFIX := pst_$(GIT_TAG)
HOST_OS := $(OS)
UNAME_S := $(shell uname -s 2>/dev/null)
UNAME_M := $(shell uname -m 2>/dev/null)
EXT :=
RELEASE_VERSION ?= $(GIT_TAG)
RELEASE_ASSET_SCRIPT := ./script/download-release-asset.sh
RELEASE_CACHE_DIR := ./.cache/release-assets/$(RELEASE_VERSION)
LOCAL_SAV_CLI_ASSET :=

ifeq ($(HOST_OS),Windows_NT)
    EXT := .exe
    LOCAL_SAV_CLI_ASSET := sav_cli_windows_x86_64.exe
else
    ifeq ($(UNAME_S),Linux)
        ifeq ($(UNAME_M),x86_64)
            LOCAL_SAV_CLI_ASSET := sav_cli_linux_x86_64
        else ifeq ($(UNAME_M),amd64)
            LOCAL_SAV_CLI_ASSET := sav_cli_linux_x86_64
        else ifeq ($(UNAME_M),aarch64)
            LOCAL_SAV_CLI_ASSET := sav_cli_linux_aarch64
        else ifeq ($(UNAME_M),arm64)
            LOCAL_SAV_CLI_ASSET := sav_cli_linux_aarch64
        endif
    endif
endif

LOCAL_SAV_CLI_PATH := $(RELEASE_CACHE_DIR)/$(LOCAL_SAV_CLI_ASSET)
WINDOWS_SAV_CLI_PATH := $(RELEASE_CACHE_DIR)/sav_cli_windows_x86_64.exe
LINUX_X86_SAV_CLI_PATH := $(RELEASE_CACHE_DIR)/sav_cli_linux_x86_64
LINUX_ARM_SAV_CLI_PATH := $(RELEASE_CACHE_DIR)/sav_cli_linux_aarch64

.PHONY: init clean-dist frontend download-sav-cli-local download-sav-cli-pub build build-pub help

# 初始化
init:
	go mod download

.PHONY: clean-dist
# 清理 dist 目录
clean-dist:
	rm -rf dist/ && mkdir -p dist/

.PHONY: frontend
# 仅构建前端
frontend:
	rm -rf assets index.html pal-conf.html
	cd web && pnpm i && pnpm build && cd ..
	git submodule update --init --recursive
	cd pal-conf && pnpm i && pnpm build && cd ..
	mv pal-conf/dist/assets/* assets/
	mv pal-conf/dist/index.html ./pal-conf.html

.PHONY: download-sav-cli-local
# 下载当前平台所需的 sav_cli 预编译文件

download-sav-cli-local:
	@if [ -z "$(LOCAL_SAV_CLI_ASSET)" ]; then \
		echo "Unsupported local platform for sav_cli download. Please set SAVE__DECODE_PATH manually."; \
		exit 1; \
	fi
	PST_RELEASE_VERSION=$(RELEASE_VERSION) $(RELEASE_ASSET_SCRIPT) $(LOCAL_SAV_CLI_ASSET) $(LOCAL_SAV_CLI_PATH)

.PHONY: download-sav-cli-pub
# 下载发行构建所需的 sav_cli 预编译文件

download-sav-cli-pub:
	PST_RELEASE_VERSION=$(RELEASE_VERSION) $(RELEASE_ASSET_SCRIPT) sav_cli_windows_x86_64.exe $(WINDOWS_SAV_CLI_PATH)
	PST_RELEASE_VERSION=$(RELEASE_VERSION) $(RELEASE_ASSET_SCRIPT) sav_cli_linux_x86_64 $(LINUX_X86_SAV_CLI_PATH)
	PST_RELEASE_VERSION=$(RELEASE_VERSION) $(RELEASE_ASSET_SCRIPT) sav_cli_linux_aarch64 $(LINUX_ARM_SAV_CLI_PATH)

.PHONY: build
# 构建当前平台版本（使用 release 中的预编译 sav_cli）
build: clean-dist frontend download-sav-cli-local
	cp $(LOCAL_SAV_CLI_PATH) ./dist/sav_cli$(EXT)
	@if [ -f ./dist/sav_cli ]; then chmod +x ./dist/sav_cli; fi
	cp example/config.yaml dist/config.yaml
	go build -ldflags="-s -w -X 'main.version=$(GIT_VERSION)'" -o ./dist/pst$(EXT) main.go

.PHONY: build-pub
# 为所有平台构建，使用 release 中的预编译 sav_cli
build-pub: clean-dist frontend download-sav-cli-pub
	mkdir -p dist/windows_x86_64 dist/linux_x86_64 dist/linux_aarch64
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X 'main.version=$(GIT_TAG)'" -o ./dist/windows_x86_64/pst.exe main.go
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.version=$(GIT_TAG)'" -o ./dist/linux_x86_64/pst main.go
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X 'main.version=$(GIT_TAG)'" -o ./dist/linux_aarch64/pst main.go

	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ./dist/pst-agent_$(GIT_TAG)_windows_x86_64.exe ./cmd/pst-agent/main.go
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./dist/pst-agent_$(GIT_TAG)_linux_x86_64 ./cmd/pst-agent/main.go
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ./dist/pst-agent_$(GIT_TAG)_linux_aarch64 ./cmd/pst-agent/main.go

	cp $(WINDOWS_SAV_CLI_PATH) dist/windows_x86_64/sav_cli.exe
	cp $(LINUX_X86_SAV_CLI_PATH) dist/linux_x86_64/sav_cli
	cp $(LINUX_ARM_SAV_CLI_PATH) dist/linux_aarch64/sav_cli
	chmod +x dist/linux_x86_64/sav_cli dist/linux_aarch64/sav_cli

	cp example/config.yaml dist/windows_x86_64/config.yaml
	cp example/config.yaml dist/linux_x86_64/config.yaml
	cp example/config.yaml dist/linux_aarch64/config.yaml
	cp script/start.bat dist/windows_x86_64/start.bat

	cd dist && zip -r $(PREFIX)_windows_x86_64.zip windows_x86_64/* && tar -czf $(PREFIX)_linux_x86_64.tar.gz linux_x86_64/* && tar -czf $(PREFIX)_linux_aarch64.tar.gz linux_aarch64/* && cd ..

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
