GO := GOTOOLCHAIN=go1.25.12 go
NPM := npm

.PHONY: build build-web build-all test lint clean run-example dev-web install check-go check-node

BINARY := driftdetect
BUILD_DIR := bin

check-go:
	@command -v go >/dev/null 2>&1 || { \
		echo "Error: Go is not installed or not on your PATH."; \
		echo ""; \
		echo "Install Go 1.25+ from https://go.dev/dl/"; \
		echo "macOS (Homebrew): brew install go"; \
		echo "Ubuntu/Debian:    sudo apt install golang-go  # or use go.dev installer for 1.25+"; \
		echo ""; \
		echo "After installing, verify with: go version"; \
		exit 127; \
	}

check-node:
	@command -v npm >/dev/null 2>&1 || { \
		echo "Error: npm is not installed or not on your PATH."; \
		echo "Install Node.js 22+ from https://nodejs.org/"; \
		exit 127; \
	}

build-web: check-node
	cd web && $(NPM) ci && $(NPM) run build
	rm -rf internal/api/webdist
	mkdir -p internal/api/webdist
	cp -r web/dist/* internal/api/webdist/

build: check-go
	$(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/driftdetect

build-all: build-web build

test: check-go
	$(GO) test ./... -v -count=1

test-cover:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	$(GO) vet ./...
	cd web && $(NPM) run lint

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html web/dist web/node_modules internal/api/webdist

dev-web:
	cd web && $(NPM) run dev

run-example: build-all
	./$(BUILD_DIR)/$(BINARY) scan \
		--state testdata/sample.tfstate \
		--provider aws \
		--output console

install:
	$(GO) install ./cmd/driftdetect
