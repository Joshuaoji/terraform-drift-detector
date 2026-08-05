GO := GOTOOLCHAIN=go1.25.12 go
NPM := npm

.PHONY: build build-web build-all test lint clean run-example dev-web install

BINARY := driftdetect
BUILD_DIR := bin

build-web:
	cd web && $(NPM) ci && $(NPM) run build
	rm -rf internal/api/webdist
	mkdir -p internal/api/webdist
	cp -r web/dist/* internal/api/webdist/

build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/driftdetect

build-all: build-web build

test:
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
