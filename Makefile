.PHONY: all build frontend run test lint clean migrate

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/LiStudioorg/gitli/internal/version.version=$(VERSION) \
           -X github.com/LiStudioorg/gitli/internal/version.commit=$(COMMIT) \
           -X github.com/LiStudioorg/gitli/internal/version.date=$(DATE)

all: build

frontend:
	cd web && npm install && npm run build

build:
	go build -ldflags "$(LDFLAGS)" -o gitli .

run:
	go run . serve

test:
	go test ./...

lint:
	gofmt -l .
	go vet ./...

migrate:
	go run . migrate

clean:
	rm -f gitli
	rm -rf web/dist
