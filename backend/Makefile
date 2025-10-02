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

build: $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS) $(STATIC_DIRS)
	mkdir dist > /dev/null 2>&1 || :
	CGO_ENABLED=0 $(AARGS) go build -C $(SRC_DIRS) -tags=embedallowed -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) " -o ../dist/$(ARCH)/ ./...

test_build: $(SRC_DIRS) $(STATIC_DIRS)
	CGO_ENABLED=0 go build -C $(SRC_DIRS) -tags=pprof -gcflags=all="-N -l" -ldflags="-X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) " -o ../tmp/srat  ./cmd/srat-server

remote_build: $(SRC_DIRS) $(STATIC_DIRS)
	mkdir -p tmp/x86_64 > /dev/null 2>&1 || :
	CGO_ENABLED=0 GOARCH=amd64 go build  -C $(SRC_DIRS) -tags=embedallowed,pprof -gcflags=all="-N -l" -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP) -X github.com/dianlight/srat/config.RollbarToken=$(ROLLBAR_CLIENT_ACCESS_TOKEN) -X github.com/dianlight/srat/config.RollbarEnvironment=$(ROLLBAR_ENVIRONMENT) " -o ../tmp/x86_64/ ./...

.PHONY: format
format:
	go tool -C $(SRC_DIRS)  gohack undo && \
	go mod -C $(SRC_DIRS) tidy && \
	go -C $(SRC_DIRS) vet ./... && \
	go tool -C $(SRC_DIRS)  testifylint --fix ./...  && \
	go -C $(SRC_DIRS) fmt ./... && \
	make patch

.PHONY: clean
clean: clean_examples
	go clean
	rm -rf dist

.PHONY: test
test: $(ALL_SRCS)
	go -C $(SRC_DIRS) tool gotest -p 1 -failfast  -timeout 120s -tags embedallowed_no  -coverpkg=./... ./...

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
	go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug -ha-url $(SUPERVISOR_URL) version

.PHONY: dev_cli
.ONESHELL:
dev_cli:
	mkdir ./tmp > /dev/null 2>&1 || :
	go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug -ha-url $(SUPERVISOR_URL) version && \
	go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug -ha-url $(SUPERVISOR_URL) upgrade -channel prerelease && \
	go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug -ha-url $(SUPERVISOR_URL) start -opt ../test/data/options.json -conf ../test/data/config.json -out ../tmp/samba.conf && \
	go run -C $(SRC_DIRS) ./cmd/srat-cli -db /tmp/testdb.db -loglevel debug -ha-url $(SUPERVISOR_URL) stop

.PHONY: dev
dev:
	#make dev_cli
	go tool -C $(SRC_DIRS) air -c .air.toml


.PHONY: update
update:
	go get -C $(SRC_DIRS) -u ./...

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

.ONESHELL:
patch:
	go tool -C $(SRC_DIRS) gohack undo
	rm -rf ${HOME}/gohack
	go tool -C $(SRC_DIRS) gohack get -vcs github.com/zarldev/goenums
	git -C ${HOME}/gohack/github.com/zarldev/goenums apply $(ROOT_DIR)/patches/goenums.patch
	go tool -C $(SRC_DIRS) gohack get -vcs github.com/jpillora/overseer
	git -C ${HOME}/gohack/github.com/jpillora/overseer apply $(ROOT_DIR)/patches/overseer.patch

.PHONY: gen_patch
gen_patch:
	mkdir patches > /dev/null 2>&1 || :
	cd ${HOME}/gohack/github.com/zarldev/goenums; git diff > $(ROOT_DIR)/patches/goenums.patch
	cd ${HOME}/gohack/github.com/jpillora/overseer; git diff > $(ROOT_DIR)/patches/overseer.patch

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
	find ./tmp/x86_64/ -name "srat*" -exec cp -v {} /mnt/remote/ \;
