GOCACHE ?= /data/home/admin1/gocache
GOTMPDIR ?= /data/home/admin1/gotmp
TMPDIR ?= /data/home/admin1/gotmp
export GOCACHE GOTMPDIR TMPDIR
export GOPROXY = https://goproxy.cn,direct

SQLC = go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0

GOFLAGS_BUILD = -trimpath -ldflags="-s -w"

PLATFORMS = \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: build dev test vet sqlc release clean $(PLATFORMS)

build:
	go build $(GOFLAGS_BUILD) -o bin/gitli ./cmd/gitli

dev:
	go run ./cmd/gitli serve

test:
	go test ./...

vet:
	go vet ./...

sqlc:
	$(SQLC) generate

# release: CGO_ENABLED=0 交叉编译各平台静态单二进制（modernc.org/sqlite 纯 Go，无需 CGO）
release:
	@mkdir -p bin
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=; [ "$$os" = windows ] && ext=.exe; \
		echo "==> building bin/gitli-$$os-$$arch$$ext"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build $(GOFLAGS_BUILD) -o bin/gitli-$$os-$$arch$$ext ./cmd/gitli; \
	done

clean:
	rm -rf bin
