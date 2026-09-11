BINARY=univelop-mcp
VERSION?=$(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS=-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)
BUILDDIR=bin

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all build test clean lint cross

all: build

build:
	@mkdir -p $(BUILDDIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILDDIR)/$(BINARY) ./cmd/univelop-mcp/

test:
	go test -v ./...

vet:
	go vet ./...

lint: vet
	@which golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

clean:
	rm -rf $(BUILDDIR)

cross:
	@for pair in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do \
		os=$${pair%/*}; arch=$${pair#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" \
			-o $(BUILDDIR)/$(BINARY)-$$os-$$arch$$ext ./cmd/univelop-mcp/; \
	done

.PHONY: run-sse
run-sse:
	$(BUILDDIR)/$(BINARY) --config config.yaml

.PHONY: run-dev
run-dev:
	$(BUILDDIR)/$(BINARY) --config config.yaml --dev-tls