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
VERSION ?= "TEST_INTERNAL"

all: PREREQUISITE build

PREREQUISITE: patch docs static

build: $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS) $(STATIC_DIRS)
	mkdir dist > /dev/null 2>&1 || :   
	cd $(SRC_DIRS);CGO_ENABLED=0 $(AARGS) go build -ldflags="-s -w -X main.SRATVersion=$(VERSION)" -o ../$(@)$(SUFFIX)

test_build: $(SRC_DIRS) $(STATIC_DIRS)
	mkdir tmp > /dev/null 2>&1 || :   
	CGO_ENABLED=0 go build -C src -gcflags=all="-N -l" -ldflags="-s -w -X main.SRATVersion=\"TEST_AIR_INTERNAL\"" -o ../tmp/srat
	CGO_ENABLED=0 GOARCH=amd64 go build -C src -gcflags=all="-N -l" -ldflags="-s -w -X main.SRATVersion=\"TEST_AIR_INTERNAL\"" -o ../tmp/srat_x86_64	


.PHONY: format
format:
	#cd $(SRC_DIRS);go run github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc4 fmt
	cd $(SRC_DIRS);go mod tidy	
	cd $(SRC_DIRS);go tool testifylint --fix ./...
	cd $(SRC_DIRS);go fmt

.PHONY: clean
clean:
	go clean
	rm -rf dist

.PHONY: test
test: $(ALL_SRCS) static
	cd $(SRC_DIRS);go tool gotest -coverpkg=./... ./...

.PHONY: test_ci
test_ci: $(ALL_SRCS)
	cd $(SRC_DIRS)/homeassistant/;go tool gotest -coverpkg=./... ./...

.PHONY: docs
docs: gen 

gen: $(SRCS)
	mkdir -p $(SRC_DIRS)/static && touch $(SRC_DIRS)/static/.nodelete
	cd $(SRC_DIRS);go generate ./...

.PHONY: dev
dev:
	#cd $(SRC_DIRS);go run . -opt ../test/data/options.json -conf ../test/data/config.json $(ARGS)
	air

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