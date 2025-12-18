TARGET_EXEC=srat
BUILD_DIR := ./dist
SRC_DIRS := ./src
FRONTEND_DIRS := ../frontend
STATIC_DIRS := ./src/web/static
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TMPDIR?=/tmp


#ALL_SRCS := $(shell find $(SRC_DIRS) -name '*.go')
ALL_SRCS := $(wildcard $(SRC_DIRS)/**/*.go)
TEST_SRCS := $(filter %_test.go, $(ALL_SRCS))
SRCS := $(filter-out %_test.go, $(ALL_SRCS))

VERSION?="$(shell git describe --tags --always --abbrev=0 --match='[0-9]*.[0-9]*.[0-9]*' 2> /dev/null)"
COMMIT_HASH="$(shell git rev-parse --short HEAD)"
BUILD_TIMESTAMP="$(shell date '+%Y-%m-%dT%H:%M:%S')"
ARCH ?= $(shell arch)
VERSION_CLEAN=$(subst ",,$(VERSION))
COMMIT_HASH_CLEAN=$(subst ",,$(COMMIT_HASH))
BUILD_TIMESTAMP_CLEAN=$(subst ",,$(BUILD_TIMESTAMP))
METADATA_C_STRING=\"version=$(VERSION_CLEAN);commit=$(COMMIT_HASH_CLEAN);build_at=$(BUILD_TIMESTAMP_CLEAN);format=elf-json\"
#METADATA_C_JSON='{\x22version\x22: \x22"$(VERSION_CLEAN)"\x22, \x22commit\x22: \x22$(COMMIT_HASH_CLEAN)\x22, \x22build_at\x22: \x22$(BUILD_TIMESTAMP_CLEAN)\x22}'
METADATA_C_JSON=\"{\x22"version"\x22:\x22"v$(VERSION_CLEAN)"\x22}\"

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
	mkdir dist > /dev/null 2>&1 || :
	CGO_ENABLED=1 \
	CGO_CFLAGS="-DELF_METADATA=$(METADATA_C_JSON)" \
	$(AARGS) go build -C $(SRC_DIRS) -tags=embedallowed -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH_CLEAN) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP_CLEAN) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) " -o ../dist/$(ARCH)/ ./...

test_build: $(SRC_DIRS) $(STATIC_DIRS)
	CGO_ENABLED=1 \
	CGO_CFLAGS="-DELF_METADATA=$(METADATA_C_JSON)" \
	go build -C $(SRC_DIRS) -tags=pprof -gcflags=all="-N -l" -ldflags="-X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH_CLEAN) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP_CLEAN) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) " -o ../tmp/$(ARCH)/ ./...

remote_build: $(SRC_DIRS) $(STATIC_DIRS)
	mkdir -p tmp/x86_64 > /dev/null 2>&1 || :
	CGO_ENABLED=1 \
	CC='zig cc -target x86_64-linux-musl' \
    CXX='zig cc -target x86_64-linux-musl' \
	GOOS=linux \
	CGO_CFLAGS="-DELF_METADATA=$(METADATA_C_JSON)" \
	GOARCH=amd64 go build  -C $(SRC_DIRS) -tags=embedallowed,pprof -gcflags=all="-N -l" -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION_CLEAN) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH_CLEAN) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP_CLEAN) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) " -o ../tmp/x86_64/ ./...

.PHONY: format
format:
	go mod -C $(SRC_DIRS) tidy && \
	go -C $(SRC_DIRS) vet ./... && \
	go tool -C $(SRC_DIRS)  testifylint --fix ./...  && \
	go -C $(SRC_DIRS) fmt ./...

.PHONY: clean
clean: clean_examples
	go clean
	rm -rf dist

.PHONY: test
test: $(ALL_SRCS)
	go -C $(SRC_DIRS) tool gotest -p 1 -failfast  -timeout 120s -tags embedallowed_no  -coverprofile=coverage.out -cover ./... && \
	go -C $(SRC_DIRS) tool cover -func=coverage.out | grep total: | awk '{print "Total coverage: " $$3}'

.PHONY: test_ci
test_ci: $(ALL_SRCS)
	cd $(SRC_DIRS)/homeassistant/;go tool gotest -failfast -coverpkg=./... ./...

.PHONY: docs
docs: gen

gen: $(SRCS)
	go generate -C $(SRC_DIRS)  ./... && \
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
	mkdir ./tmp > /dev/null 2>&1 || :
	# version command doesn't need -db flag
	go run -C $(SRC_DIRS) ./cmd/srat-cli -loglevel debug version

.PHONY: dev_cli_version
.ONESHELL:
dev_cli_version:
	mkdir ./tmp > /dev/null 2>&1 || :
	go run -C $(SRC_DIRS) ./cmd/srat-cli -loglevel debug version

.PHONY: dev_cli_upgrade
.ONESHELL:
dev_cli_upgrade:
	mkdir ./tmp > /dev/null 2>&1 || :
	go run -C $(SRC_DIRS) ./cmd/srat-cli -loglevel debug upgrade -channel prerelease

.PHONY: dev_cli_stop
.ONESHELL:
dev_cli_stop:
	mkdir ./tmp > /dev/null 2>&1 || :
	#go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug start -out ../tmp/samba.conf && \
	go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug stop

.PHONY: dev
dev:
	SRAT_DEVMODE=true go tool -C $(SRC_DIRS) air -c .air.toml


.PHONY: update
update:
	go get -C $(SRC_DIRS) -u -t ./... && \
	go -C $(SRC_DIRS) mod vendor && \
	make patch format

.PHONY: show_update
show_update:
	go -C $(SRC_DIRS) list -mod=mod -m -u all 

.PHONY: examples
examples: example_tlog

.PHONY: example_tlog
example_tlog:
	@echo "Building and running tlog example..."
	cd $(SRC_DIRS)/tlog/example && go build -o tlog-example . && ./tlog-example

.PHONY: test_examples
test_examples:
	@echo "Running example tests..."
	cd $(SRC_DIRS)/tlog/example && go test -run Example -v

.PHONY: clean_examples
clean_examples:
	@echo "Cleaning example binaries..."
	rm -f $(SRC_DIRS)/tlog/example/tlog-example


$(FRONTEND_DIRS)/node_modules:
	cd $(FRONTEND_DIRS); bun install

static: $(FRONTEND_DIRS)/node_modules
	rm -rf $(SRC_DIRS)/static
	#cd $(FRONTEND_DIRS);bun run gen
	cd $(FRONTEND_DIRS);bun run version -o $(VERSION)
	cd $(FRONTEND_DIRS);bun run build -o ../backend/$(STATIC_DIRS)

.PHONY: patch
patch:
	@echo "Applying patches to vendored dependencies..."
	@if ! command -v patch >/dev/null 2>&1; then \
		echo "ERROR: 'patch' command not found. Please install patch utility."; \
		exit 1; \
	fi
	go -C $(SRC_DIRS) mod vendor
	@find $(ROOT_DIR)/src/vendor -name "*.rej" -delete 2>/dev/null || true
	@if [ -f "$(ROOT_DIR)/patches/goenums.patch" ] && [ -d "$(ROOT_DIR)/src/vendor/github.com/zarldev/goenums" ]; then \
		cd $(ROOT_DIR)/src/vendor/github.com/zarldev/goenums && patch -p1 < $(ROOT_DIR)/patches/goenums.patch; \
	fi
	@if [ -f "$(ROOT_DIR)/patches/overseer.patch" ] && [ -d "$(ROOT_DIR)/src/vendor/github.com/jpillora/overseer" ]; then \
		cd $(ROOT_DIR)/src/vendor/github.com/jpillora/overseer  && patch -p1 < $(ROOT_DIR)/patches/overseer.patch; \
	fi
	@if [ -f "$(ROOT_DIR)/patches/gorm-generics.patch" ] && [ -d "$(ROOT_DIR)/src/vendor/gorm.io/gorm" ]; then \
		cd $(ROOT_DIR)/src/vendor/gorm.io/gorm && patch -p1 < $(ROOT_DIR)/patches/gorm-generics.patch; \
	fi
	@find $(ROOT_DIR)/src/vendor -name "*.rej" -delete 2>/dev/null || true
	@echo "Patches applied successfully"

.PHONY: gen_patch
gen_patch:
	@echo "Generating patches from vendored dependencies..."
	@mkdir -p $(ROOT_DIR)/patches
	@echo "Note: Generating patches from vendored code. For future updates:"
	@echo "  1. Manually edit files in $(ROOT_DIR)/src/vendor/github.com/{library}/"
	@echo "  2. Use: diff -Naur original_version patched_version > patches/{name}.patch"
	@echo "  3. Or initialize git repos in vendor directories for git diff"

/mnt/remote/config.db3: mount_remote
	[ -f /mnt/remote/config.db3 ] || echo "No remote mount found use make mount_remote"

.PHONY: mount_remote
mount_remote:
	umount -l /mnt/remote > /dev/null 2>&1 || :
	mkdir -p /mnt/remote > /dev/null 2>&1 || :
	sshfs -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3 ${HOMEASSISTANT_IP}:/addon_configs/local_sambanas2 /mnt/remote

.PHONY: build_remote
build_remote: /mnt/remote/config.db3
	cd $(FRONTEND_DIRS);bun run build -o ../backend/$(STATIC_DIRS);cd - && \
	$(MAKE) remote_build && \
	cp -v ./tmp/x86_64/srat-cli /mnt/remote/upgrade/ && \
	cp -v ./tmp/x86_64/srat-server /mnt/remote/upgrade/
