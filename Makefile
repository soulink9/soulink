# soulink-local 构建与本地发版：六平台交叉编译 + 归档 + checksums。
# 对齐 open-code-review 的 Makefile 方案（纯 go build 循环，不依赖 goreleaser）。

BINARY    := soulink
DIST_DIR  := dist
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell TZ='Asia/Shanghai' date '+%Y-%m-%dT%H:%M+08:00')

# -ldflags 按空格分词，值内不能含空格（DATE 形如 2026-10-09T21:30+08:00，无空格，安全）
LD_FLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
# 换牌/自托管发布：设了 SOULINK_DEFAULT_SERVER 才注入覆盖，不设沿用源码内置值（避免两份真相）
ifdef SOULINK_DEFAULT_SERVER
LD_FLAGS += -X main.defaultServer=$(SOULINK_DEFAULT_SERVER)
endif

.PHONY: build build-all dist clean test version-info

build: ## 本机平台 → bin/soulink
	go build -trimpath -ldflags "$(LD_FLAGS)" -o bin/$(BINARY) ./cmd/soulink

build-all: ## 六平台裸二进制 → dist/
	@mkdir -p $(DIST_DIR)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out="$(DIST_DIR)/$(BINARY)-$$os-$$arch$$ext"; \
		echo "  $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -trimpath -ldflags "$(LD_FLAGS)" -o "$$out" ./cmd/soulink || exit 1; \
	done

# dist 保留裸二进制不删：npm 平台包由 scripts/publish-npm.sh 从中取二进制。
# 裸二进制不进 GitHub Release 资产——release.yml 显式列举了上传的文件名。
dist: clean build-all ## 六平台归档 + checksums → dist/（unix tar.gz / windows zip，各含 README）
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		bin="$(BINARY)-$$os-$$arch$$ext"; \
		pkg="$(BINARY)_$(VERSION)_$${os}_$${arch}"; \
		if [ "$$os" = "windows" ]; then \
			zip -q -j "$(DIST_DIR)/$$pkg.zip" "$(DIST_DIR)/$$bin" README.md || exit 1; \
		else \
			tar -czf "$(DIST_DIR)/$$pkg.tar.gz" -C $(DIST_DIR) "$$bin" -C .. README.md || exit 1; \
		fi; \
	done
	@cd $(DIST_DIR) && shasum -a 256 *.tar.gz *.zip | sort > checksums.txt
	@echo "=== 产物（$(VERSION) / $(COMMIT)）==="
	@ls -lh $(DIST_DIR)

clean:
	rm -rf $(DIST_DIR)

test:
	go test ./... -count=1

version-info:
	@echo "version=$(VERSION) commit=$(COMMIT) date=$(DATE)"
	@echo "ldflags: $(LD_FLAGS)"
