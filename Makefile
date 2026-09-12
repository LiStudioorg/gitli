GOCACHE ?= /data/home/admin1/gocache
GOTMPDIR ?= /data/home/admin1/gotmp
export GOCACHE GOTMPDIR
export GOPROXY = https://goproxy.cn,direct

SQLC = go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0

.PHONY: build dev test sqlc vet

build:
	go build -o bin/gitli ./cmd/gitli

dev:
	go run ./cmd/gitli serve

test:
	go test ./...

vet:
	go vet ./...

sqlc:
	$(SQLC) generate
