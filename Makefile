.PHONY: all build build-with-frontend frontend run test lint clean migrate release-check release-snapshot

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
           -X github.com/LiStudioorg/gitli/internal/version.version=$(VERSION) \
           -X github.com/LiStudioorg/gitli/internal/version.commit=$(COMMIT) \
           -X github.com/LiStudioorg/gitli/internal/version.date=$(DATE)

# 纯静态编译：CGO_ENABLED=0 + -trimpath + 去符号/调试信息
GOFLAGS := -trimpath

all: build

frontend:
	cd web-nuxt && npm install && npm run build

# 默认 build 只编译 Go：前端产物（web/dist）随仓库提交，日常改后端不必等 Nuxt 构建。
# 需要重建前端时用 `make build-with-frontend`。
build:
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o gitli .

build-with-frontend: frontend
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o gitli .

run:
	go run . serve

test:
	CGO_ENABLED=0 go test ./...

lint:
	gofmt -l .
	go vet ./...

migrate:
	go run . migrate

release-check:
	goreleaser check

release-snapshot:
	goreleaser build --snapshot --clean

clean:
	rm -f gitli
	rm -rf web/dist dist web-nuxt/.output
