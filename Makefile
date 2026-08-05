GO := go
APP_NAME := api-server
SERVER_BIN := ./out/$(APP_NAME)
LICENSE_GEN_BIN := ./out/license-gen
BUILD_TS := $(shell date +%s)
LDFLAGS := -s -w \
	-X api-first/cmd/server.Version=$(VERSION) \
	-X api-first/cmd/server.BuildAt=$(BUILD_TS) \
	-X api-first/pkg/license.buildTimestamp=$(BUILD_TS)

VERSION ?= dev

.PHONY: all build clean test license-gen

all: build

# Build the main server binary with build timestamp injected for clock tampering detection
build:
	@mkdir -p ./out
	@echo "Building $(APP_NAME) with build timestamp $(BUILD_TS)..."
	$(GO) build -ldflags="$(LDFLAGS)" -o $(SERVER_BIN) ./cmd/server

# Build the license generator CLI
license-gen:
	@mkdir -p ./out
	@echo "Building license-gen..."
	$(GO) build -o $(LICENSE_GEN_BIN) ./cmd/license-gen

# Run tests
test:
	$(GO) test -v ./...

# Clean build artifacts
clean:
	rm -rf ./out

# Generate a test license (for development only)
dev-license: license-gen
	@echo "Generating development license..."
	@export SIGNING_PRIVATE_KEY="$$(cat cmd/license-gen/private.pem)" && \
	$(LICENSE_GEN_BIN) -customer="DEV-TEAM" -expiry=365 -hwid="" -features="ha-mode,audit-log" -max-instances=5 -out=./dev.lic