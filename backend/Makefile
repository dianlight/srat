TARGET_EXEC=srat
BUILD_DIR := ./dist
SRC_DIRS := ./src
FRONTEND_DIRS := ../frontend
STATIC_DIRS := ./src/web/static
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))


#ALL_SRCS := $(shell find $(SRC_DIRS) -name '*.go')
ALL_SRCS := $(wildcard $(SRC_DIRS)/**/*.go)
TEST_SRCS := $(filter %_test.go, $(ALL_SRCS))
SRCS := $(filter-out %_test.go, $(ALL_SRCS))

VERSION?="$(shell git describe --tags --always --abbrev=0 --match='[0-9]*.[0-9]*.[0-9]*' 2> /dev/null)"
COMMIT_HASH="$(shell git rev-parse --short HEAD)"
BUILD_TIMESTAMP="$(shell date '+%Y-%m-%dT%H:%M:%S')"
ARCH ?= $(shell arch)

all: PREREQUISITE build

PREREQUISITE: patch docs static

build: $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS) $(STATIC_DIRS)
	mkdir dist > /dev/null 2>&1 || :
	CGO_ENABLED=0 $(AARGS) go build -C $(SRC_DIRS) -tags=embedallowed -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP)" -o ../dist/$(ARCH)/ ./...

test_build: $(SRC_DIRS) $(STATIC_DIRS)
	mkdir -p tmp/x86_64 > /dev/null 2>&1 || :
	CGO_ENABLED=0 go build -C $(SRC_DIRS) -tags=embedallowed_no -gcflags=all="-N -l" -ldflags="-X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP)" -o ../tmp/srat  ./cmd/srat-server && \
	CGO_ENABLED=0 GOARCH=amd64 go build  -C $(SRC_DIRS) -tags=embedallowed -gcflags=all="-N -l" -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP)" -o ../tmp/x86_64/ ./... && \
	[ -f /mnt/remote/bootconfig.json ] && find ./tmp/x86_64/ -name "srat*" -exec cp -v {} /mnt/remote/ \; || echo "No remote mount found use make mount_remote"

.PHONY: format
format:
	go tool -C $(SRC_DIRS)  gohack undo
	go mod -C $(SRC_DIRS) tidy
	go tool -C $(SRC_DIRS)  testifylint --fix ./...
	go -C $(SRC_DIRS)  fmt
	make patch

.PHONY: clean
clean:
	go clean
	rm -rf dist

.PHONY: test
test: $(ALL_SRCS)
	cd $(SRC_DIRS);GOMAXPROCS=1 go tool gotest -failfast -coverpkg=./... ./...

.PHONY: test_ci
test_ci: $(ALL_SRCS)
	cd $(SRC_DIRS)/homeassistant/;go tool gotest -failfast -coverpkg=./... ./...

.PHONY: docs
docs: gen

gen: $(SRCS)
	go generate -C $(SRC_DIRS)  ./... && \
	go run -C $(SRC_DIRS) ./cmd/srat-openapi -out ../docs

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
	make dev_cli
	go tool -C $(SRC_DIRS) air -c .air.toml


.PHONY: update
update:
	go get -C $(SRC_DIRS) -u ./...


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
	go tool -C $(SRC_DIRS) gohack get -vcs github.com/samber/slog-http
	git -C ${HOME}/gohack/github.com/samber/slog-http apply $(ROOT_DIR)/patches/slog-http.patch
	go tool -C $(SRC_DIRS) gohack get -vcs github.com/jpillora/overseer
	git -C ${HOME}/gohack/github.com/jpillora/overseer apply $(ROOT_DIR)/patches/overseer.patch

gen_patch:
	mkdir patches > /dev/null 2>&1 || :
	cd ${HOME}/gohack/github.com/samber/slog-http; git diff > $(ROOT_DIR)/patches/slog-http.patch
	cd ${HOME}/gohack/github.com/jpillora/overseer; git diff > $(ROOT_DIR)/patches/overseer.patch

 /mnt/remote/bootconfig.json: mount_remote
	[ -f /mnt/remote/bootconfig.json ] || echo "No remote mount found use make mount_remote"

.PHONY: mount_remote
mount_remote:
	umount -l /mnt/remote > /dev/null 2>&1 || :
	mkdir -p /mnt/remote > /dev/null 2>&1 || :
	sshfs -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3 ${HOMEASSISTANT_IP}:/addon_configs/local_sambanas2 /mnt/remote

.PHONY: build_remote
build_remote: /mnt/remote/bootconfig.json
	cd $(FRONTEND_DIRS);bun run build -o ../backend/$(STATIC_DIRS);cd - && \
	$(MAKE) test_build
