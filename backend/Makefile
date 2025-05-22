TARGET_EXEC=srat
BUILD_DIR := ./dist
SRC_DIRS := ./src
FRONTEND_DIRS := ../frontend
STATIC_DIRS := ./src/static
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))


#ALL_SRCS := $(shell find $(SRC_DIRS) -name '*.go')
ALL_SRCS := $(wildcard $(SRC_DIRS)/**/*.go)  
TEST_SRCS := $(filter %_test.go, $(ALL_SRCS))
SRCS := $(filter-out %_test.go, $(ALL_SRCS))

VERSION?="$(shell git describe --tags --always --abbrev=0 --match='v[0-9]*.[0-9]*.[0-9]*' 2> /dev/null | sed 's/^.//')"
COMMIT_HASH="$(shell git rev-parse --short HEAD)"
BUILD_TIMESTAMP="$(shell date '+%Y-%m-%dT%H:%M:%S')"

all: PREREQUISITE build

PREREQUISITE: patch docs static

build: $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS) $(STATIC_DIRS)
	mkdir dist > /dev/null 2>&1 || :   
	cd $(SRC_DIRS);CGO_ENABLED=0 $(AARGS) go build -tags=embedallowed -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP)" -o ../$(@)$(SUFFIX)

test_build: $(SRC_DIRS) $(STATIC_DIRS)
	mkdir tmp > /dev/null 2>&1 || :   
	CGO_ENABLED=0 go build -C src -tags=embedallowed_no -gcflags=all="-N -l" -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP)" -o ../tmp/srat
	CGO_ENABLED=0 GOARCH=amd64 go build  -C src -tags=embedallowed -gcflags=all="-N -l" -ldflags="-s -w -X github.com/dianlight/srat/config.Version=$(VERSION) -X github.com/dianlight/srat/config.CommitHash=$(COMMIT_HASH) -X github.com/dianlight/srat/config.BuildTimestamp=$(BUILD_TIMESTAMP)" -o ../tmp/srat_x86_64	
	[ -f /mnt/remote/bootconfig.json ] && cp tmp/srat_x86_64 /mnt/remote/srat_x86_64 || echo "No remote mount found use make mount_remote"

.PHONY: format
format:
	cd $(SRC_DIRS);go tool gohack undo
	cd $(SRC_DIRS);go mod tidy	
	cd $(SRC_DIRS);go tool testifylint --fix ./...
	cd $(SRC_DIRS);go fmt
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
	mkdir -p $(SRC_DIRS)/static && touch $(SRC_DIRS)/static/.nodelete
	cd $(SRC_DIRS);go generate ./...

.PHONY: dev
dev:
	#cd $(SRC_DIRS);go run . -opt ../test/data/options.json -conf ../test/data/config.json $(ARGS)
	cd $(SRC_DIRS);go tool air

.PHONY: update
update:
	cd $(SRC_DIRS);go get -u ./...


$(FRONTEND_DIRS)/node_modules:
	cd $(FRONTEND_DIRS); bun install	

static: $(FRONTEND_DIRS)/node_modules
	rm -rf $(SRC_DIRS)/static
	#cd $(FRONTEND_DIRS);bun run gen
	cd $(FRONTEND_DIRS);bun run version -o $(VERSION)
	cd $(FRONTEND_DIRS);bun run build -o ../backend/$(SRC_DIRS)/static

patch:
	cd $(SRC_DIRS);go tool gohack undo
	rm -rf ${HOME}/gohack
	cd $(SRC_DIRS);go tool gohack get -vcs github.com/samber/slog-http
	cd ${HOME}/gohack/github.com/samber/slog-http; git apply $(ROOT_DIR)/patches/slog-http.patch
	cd $(SRC_DIRS);go tool gohack get -vcs github.com/jpillora/overseer	
	cd ${HOME}/gohack/github.com/jpillora/overseer; git apply $(ROOT_DIR)/patches/overseer.patch

gen_patch:
	mkdir patches > /dev/null 2>&1 || :  
	cd ${HOME}/gohack/github.com/samber/slog-http; git diff > $(ROOT_DIR)/patches/slog-http.patch
	cd ${HOME}/gohack/github.com/jpillora/overseer; git diff > $(ROOT_DIR)/patches/overseer.patch


.PHONY: mount_remote
mount_remote:
	umount -l /mnt/remote > /dev/null 2>&1 || :
	mkdir -p /mnt/remote > /dev/null 2>&1 || :
	sshfs -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3 root@192.168.0.68:/addon_configs/local_sambanas /mnt/remote
