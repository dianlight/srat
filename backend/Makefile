# Configuration variables
TARGET_EXEC=srat-server
SRC_DIRS := ./src
FRONTEND_DIRS := ../frontend
STATIC_DIRS := ./src/web/static
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TMPDIR?=/tmp

# CGO and build tools configuration
CGO_ENABLED?=0
USE_ZIG?=0
PPROF?=0


#ALL_SRCS := $(shell find $(SRC_DIRS) -name '*.go')
ALL_SRCS := $(wildcard $(SRC_DIRS)/**/*.go) $(SRC_DIRS)/config/metadata_constants.go
TEST_SRCS := $(filter %_test.go, $(ALL_SRCS))
SRCS := $(filter-out %_test.go, $(ALL_SRCS))

VERSION?="$(shell git describe --tags --always --abbrev=0 --match='[0-9]*.[0-9]*.[0-9]*' 2> /dev/null | python3 -c 'import sys,re; v=sys.stdin.read().strip(); m=re.match(r"^(\d+\.\d+\.\d+)(?:-([a-z]+)\.(\d+))?$$",v); print(f"{m[1]}-{m[2]}.{int(m[3])+1}" if m and m[2] else f"{m[1]}-dev.0" if m else v)')"
COMMIT_HASH="$(shell git rev-parse --short HEAD)"
BUILD_TIMESTAMP="$(shell date '+%Y-%m-%dT%H:%M:%S')"
ARCH ?= $(shell arch)
BUILD_DIR := ./dist/$(ARCH)
VERSION_CLEAN=$(subst ",,$(VERSION))
COMMIT_HASH_CLEAN=$(subst ",,$(COMMIT_HASH))
BUILD_TIMESTAMP_CLEAN=$(subst ",,$(BUILD_TIMESTAMP))
METADATA_C_STRING=\"version=$(VERSION_CLEAN);commit=$(COMMIT_HASH_CLEAN);build_at=$(BUILD_TIMESTAMP_CLEAN);format=elf-json\"
METADATA_C_JSON=\"{\x22"version"\x22:\x22"v$(VERSION_CLEAN)"\x22}\"

GIST_TOKEN?=""	
# Telemetry configuration from environment variables
ROLLBAR_CLIENT_ACCESS_TOKEN?=""
# Auto-determine ROLLBAR_ENVIRONMENT based on VERSION
ROLLBAR_ENVIRONMENT?=$(shell \
	if echo $(VERSION) | grep -q "\-dev\."; then \
		echo "development"; \
	elif echo $(VERSION) | grep -q "\-rc\."; then \
		echo "prerelease"; \
	else \
		echo "production"; \
	fi)

all: PREREQUISITE build

PREREQUISITE: patch docs static

build: static $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS) $(STATIC_DIRS)
	@mkdir dist > /dev/null 2>&1 || :
	@echo "Building $(TARGET_EXEC) for architecture $(ARCH) [tags=p$(if $(filter 1,$(PPROF)),prof),embedallowed]..."
	CGO_ENABLED=$(CGO_ENABLED) \
	$(if $(filter 1,$(USE_ZIG)),CC='zig cc -target $(ARCH)-linux-musl' \,)
	CGO_CFLAGS="-DELF_METADATA=$(METADATA_C_JSON)" \
	$(AARGS) go build -C $(SRC_DIRS) -tags=$(if $(filter 1,$(PPROF)),pprof),embedallowed -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH_CLEAN) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP_CLEAN) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) -X github.com/dianlight/srat/config.GistToken=$(GIST_TOKEN) " -o ../dist/$(ARCH)/ ./...
	@echo "Verifying embedded version in $(TARGET_EXEC)..."
	@EMBEDDED_VERSION=$$(../scripts/extract-binary-version.sh $(BUILD_DIR)/$(TARGET_EXEC) 2>&1); \
	if [ $$? -eq 0 ]; then \
		if [ "$$EMBEDDED_VERSION" = "v$(VERSION_CLEAN)" ]; then \
			echo "✓ Version verified: $$EMBEDDED_VERSION"; \
		else \
			echo "✗ Version mismatch! Expected: v$(VERSION_CLEAN), Got: $$EMBEDDED_VERSION"; \
			exit 1; \
		fi; \
	else \
		echo "⚠ Warning: Could not extract version from binary: $$EMBEDDED_VERSION"; \
	fi

#test_build: $(SRC_DIRS) $(STATIC_DIRS)
#	CGO_ENABLED=$(CGO_ENABLED) \
#	$(if $(filter 1,$(USE_ZIG)),CC='zig cc -target $(ARCH)-linux-musl' \,)
#	CGO_CFLAGS="-DELF_METADATA=$(METADATA_C_JSON)" \
#	go build -C $(SRC_DIRS) -tags=pprof -gcflags=all="-N -l" -ldflags="-X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH_CLEAN) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP_CLEAN) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) " -o ../tmp/$(ARCH)/ ./...
#remote_build: $(SRC_DIRS) $(STATIC_DIRS)
#	mkdir -p tmp/x86_64 > /dev/null 2>&1 || :
#	CGO_ENABLED=$(CGO_ENABLED) \
#	$(if $(filter 1,$(USE_ZIG)),CC='zig cc -target x86_64-linux-musl' \,)
#	GOOS=linux \
#	CGO_CFLAGS="-DELF_METADATA=$(METADATA_C_JSON)" \
#	GOARCH=amd64 go build  -C $(SRC_DIRS) -tags=embedallowed,pprof -gcflags=all="-N -l" -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH_CLEAN) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP_CLEAN) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) " -o ../tmp/x86_64/ ./...

.PHONY: format
format:
	@go mod -C $(SRC_DIRS) tidy && \
	go -C $(SRC_DIRS) vet ./... && \
	go tool -C $(SRC_DIRS)  testifylint --fix ./...  && \
	go -C $(SRC_DIRS) fmt ./...

.PHONY: clean
clean: clean_examples
	@go clean
	@rm -rf dist

.PHONY: test
test: $(ALL_SRCS)
	@go -C $(SRC_DIRS) tool gotest -p 1 -failfast  -timeout 120s -tags embedallowed_no  -coverprofile=coverage.out -cover ./... && \
	go -C $(SRC_DIRS) tool cover -func=coverage.out | grep total: | awk '{print "Total coverage: " $$3}'

.PHONY: test_ci
test_ci: $(ALL_SRCS)
	cd $(SRC_DIRS)/homeassistant/;go tool gotest -failfast -coverpkg=./... ./...

.PHONY: $(SRC_DIRS)/config/metadata_constants.go 
$(SRC_DIRS)/config/metadata_constants.go:
	@echo "Generating metadata constants..."
	@echo "// Code generated by Makefile; DO NOT EDIT." > $(SRC_DIRS)/config/metadata_constants.go
	@echo "//go:build !cgo" >> $(SRC_DIRS)/config/metadata_constants.go
	@echo "" >> $(SRC_DIRS)/config/metadata_constants.go
	@echo "package config" >> $(SRC_DIRS)/config/metadata_constants.go
	@echo "const MetadataJSON = \"SRAT_METADATA:{\\\"version\\\": \\\"v$(VERSION_CLEAN)\\\"}\"" >> $(SRC_DIRS)/config/metadata_constants.go

.PHONY: test_metadata
test_metadata: $(SRC_DIRS)/config/metadata_constants.go
	@echo "=== Testing Metadata Embedding ==="
	@echo "Version: $(VERSION_CLEAN)"
	@echo "Metadata JSON: $(METADATA_C_JSON)"
	@echo ""
	@echo "Building srat-openapi with CGO enabled..."
	@mkdir -p $(TMPDIR)/metadata_test
	@CGO_ENABLED=1 \
		CGO_CFLAGS="-DELF_METADATA=$(METADATA_C_JSON)" \
		go build -C $(SRC_DIRS) \
		-ldflags="-X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH_CLEAN) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP_CLEAN)" \
		-o $(TMPDIR)/metadata_test/srat-openapi-cgo \
		./cmd/srat-openapi
	@echo "✓ CGO-enabled build completed"
	@echo ""
	@echo "=== Verifying Embedded Metadata ==="
	@EMBEDDED_VERSION=$$(../scripts/extract-binary-version.sh $(TMPDIR)/metadata_test/srat-openapi-cgo 2>&1); \
	if [ $$? -eq 0 ]; then \
		echo "✓ Extracted version: $$EMBEDDED_VERSION"; \
		echo "Expected version: v$(VERSION_CLEAN)"; \
		if [ "$$EMBEDDED_VERSION" = "v$(VERSION_CLEAN)" ]; then \
			echo "✓ Version matches!"; \
		else \
			echo "✗ Version mismatch!"; \
			rm -rf $(TMPDIR)/metadata_test; \
			exit 1; \
		fi; \
	else \
		echo "✗ Failed to extract version: $$EMBEDDED_VERSION"; \
		rm -rf $(TMPDIR)/metadata_test; \
		exit 1; \
	fi
	@echo ""
	@echo "=== Testing No-CGO Build ==="
	@echo "Building srat-openapi with CGO disabled..."
	@if CGO_ENABLED=0 go build -C $(SRC_DIRS) \
		-ldflags="-X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN)" \
		-o $(TMPDIR)/metadata_test/srat-openapi-nocgo ./cmd/srat-openapi 2>&1; then \
		echo "✓ No-CGO build succeeded"; \
		echo ""; \
		echo "Extracting SRAT metadata..."; \
		EXTRACTED_VERSION=$$(../scripts/extract-binary-version.sh $(TMPDIR)/metadata_test/srat-openapi-nocgo 2>&1); \
		if [ $$? -eq 0 ]; then \
			echo "✓ Extracted version: $$EXTRACTED_VERSION"; \
			echo "Expected version: v$(VERSION_CLEAN)"; \
			if [ "$$EXTRACTED_VERSION" = "v$(VERSION_CLEAN)" ]; then \
				echo "✓ Version matches!"; \
			else \
				echo "✗ Metadata version mismatch"; \
				rm -rf $(TMPDIR)/metadata_test; \
				exit 1; \
			fi; \
		else \
			echo "✗ Failed to extract version: $$EXTRACTED_VERSION"; \
			rm -rf $(TMPDIR)/metadata_test; \
			exit 1; \
		fi; \
		echo ""; \
		echo "Note: Runtime metadata is available via GetBinaryVersion() and GetCurrentBinaryVersion()."; \
	else \
		echo "✗ No-CGO build failed unexpectedly"; \
		CGO_ENABLED=0 go build -C $(SRC_DIRS) \
			-ldflags="-X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN)" \
			-o $(TMPDIR)/metadata_test/srat-openapi-nocgo ./cmd/srat-openapi; \
		rm -rf $(TMPDIR)/metadata_test; \
		exit 1; \
	fi
	@echo ""
	@echo "=== All Metadata Tests Passed ==="
	@rm -rf $(TMPDIR)/metadata_test

.PHONY: docs
docs: gen

gen: $(SRCS)
	@go generate -C $(SRC_DIRS)  ./... && \
	go run -C $(SRC_DIRS) ./cmd/srat-openapi -out ../docs && \
	bunx @redocly/cli@latest lint ./docs/openapi.yaml


.PHONY: gosec security
# Run Go security checks with gosec
gosec:
	@echo "Running gosec security scan..."
	@GOSEC_BIN=$$(command -v gosec 2>/dev/null || echo $$(go env GOPATH)/bin/gosec); \
	if [ ! -x "$$GOSEC_BIN" ]; then \
		echo "Installing gosec..."; \
		GOBIN=$$(go env GOPATH)/bin GO111MODULE=on go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		GOSEC_BIN=$$(go env GOPATH)/bin/gosec; \
	fi; \
	cd $(SRC_DIRS); "$$GOSEC_BIN" -exclude-generated -severity high -confidence high ./...

# Alias
security: gosec

.PHONY: dev_version
.ONESHELL:
dev_version:
	@mkdir ./tmp > /dev/null 2>&1 || :
	# version command doesn't need -db flag
	go run -C $(SRC_DIRS) ./cmd/srat-cli -loglevel debug version

.PHONY: dev_cli_version
.ONESHELL:
dev_cli_version:
	@mkdir ./tmp > /dev/null 2>&1 || :
	go run -C $(SRC_DIRS) ./cmd/srat-cli -loglevel debug version

.PHONY: dev_cli_upgrade
.ONESHELL:
dev_cli_upgrade:
	@mkdir ./tmp > /dev/null 2>&1 || :
	go run -C $(SRC_DIRS) ./cmd/srat-cli -loglevel debug upgrade -channel prerelease

.PHONY: dev_cli_stop
.ONESHELL:
dev_cli_stop:
	@mkdir ./tmp > /dev/null 2>&1 || :
	#go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug start -out ../tmp/samba.conf && \
	go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug stop

.PHONY: dev
dev:
	@SRAT_DEVMODE=true go tool -C $(SRC_DIRS) air -c .air.toml


.PHONY: update
update:
	@go -C $(SRC_DIRS) tool github.com/oligot/go-mod-upgrade && \
	go -C $(SRC_DIRS) mod vendor && \
	make patch format


$(FRONTEND_DIRS)/node_modules:
	@cd $(FRONTEND_DIRS); bun install

static: $(FRONTEND_DIRS)/node_modules
	@rm -rf ../backend/$(STATIC_DIRS)/* && \
	[ -f ../backend/$(STATIC_DIRS)/.keep ] || touch ../backend/$(STATIC_DIRS)/.keep && \
	cd $(FRONTEND_DIRS) && \
	bun run version -o $(VERSION) && \
	bun run build -o ../backend/$(STATIC_DIRS)

.PHONY: patch
patch:
	@set -e; \
	echo "Applying patches to vendored dependencies..."; \
	if ! command -v patch >/dev/null 2>&1; then \
		echo "ERROR: 'patch' command not found. Please install patch utility."; \
		exit 1; \
	fi; \
	go -C $(SRC_DIRS) mod vendor; \
	find $(ROOT_DIR)/src/vendor -name "*.rej" -delete 2>/dev/null || true; \
	if [ -f "$(ROOT_DIR)/patches/goenums.patch" ] && [ -d "$(ROOT_DIR)/src/vendor/github.com/zarldev/goenums" ]; then \
		echo "Applying goenums.patch..."; \
		cd $(ROOT_DIR)/src/vendor/github.com/zarldev/goenums && patch -p1 < $(ROOT_DIR)/patches/goenums.patch || (echo "ERROR: Failed to apply goenums.patch"; exit 1); \
	fi; \
	if [ -f "$(ROOT_DIR)/patches/gorm-generics.patch" ] && [ -d "$(ROOT_DIR)/src/vendor/gorm.io/gorm" ]; then \
		echo "Applying gorm-generics.patch..."; \
		cd $(ROOT_DIR)/src/vendor/gorm.io/gorm && patch -p1 < $(ROOT_DIR)/patches/gorm-generics.patch || (echo "ERROR: Failed to apply gorm-generics.patch"; exit 1); \
	fi; \
	find $(ROOT_DIR)/src/vendor -name "*.rej" -delete 2>/dev/null || true; \
	echo "Patches applied successfully"

.PHONY: gen_patch
gen_patch:
	@echo "Generating patches from vendored dependencies..."
	@mkdir -p $(ROOT_DIR)/patches
	@echo "Note: Generating patches from vendored code. For future updates:"
	@echo "  1. Manually edit files in $(ROOT_DIR)/src/vendor/github.com/{library}/"
	@echo "  2. Use: diff -Naur original_version patched_version > patches/{name}.patch"
	@echo "  3. Or initialize git repos in vendor directories for git diff"

/mnt/remote/config.db3: mount_remote
	@[ -f /mnt/remote/config.db3 ] || echo "No remote mount found use make mount_remote"

.PHONY: mount_remote
mount_remote:
	@umount -l /mnt/remote > /dev/null 2>&1 || :
	@mkdir -p /mnt/remote > /dev/null 2>&1 || :
	@sshfs -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3 ${HOMEASSISTANT_IP}:/addon_configs/local_sambanas2 /mnt/remote

.PHONY: build_remote
build_remote: /mnt/remote/config.db3
	@rm -rf ./dist/x86_64 || :
	@$(MAKE) build AARGS="GOARCH=amd64" VERSION=$(VERSION) ARCH="x86_64" PPROF=1 && \
    cp -v ./dist/x86_64/srat-cli /mnt/remote/upgrade/ && \
    cp -v ./dist/x86_64/srat-server /mnt/remote/upgrade/ && \
    echo "Remote build and deployment completed."