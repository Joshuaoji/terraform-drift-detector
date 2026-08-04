GO := GOTOOLCHAIN=go1.25.12 go

.PHONY: build test lint clean run-example

BINARY := driftdetect
BUILD_DIR := bin

build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/driftdetect

test:
	$(GO) test ./... -v -count=1

test-cover:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

run-example: build
	./$(BUILD_DIR)/$(BINARY) scan \
		--state testdata/sample.tfstate \
		--provider aws \
		--output console

install:
	$(GO) install ./cmd/driftdetect
