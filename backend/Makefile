TARGET_EXEC=srat
BUILD_DIR := ./dist
SRC_DIRS := ./src
FRONTEND_DIRS := ../frontend
STATIC_DIRS := $(FRONTEND_DIRS)/out

#ALL_SRCS := $(shell find $(SRC_DIRS) -name '*.go')
ALL_SRCS := $(wildcard $(SRC_DIRS)/**/*.go)  
TEST_SRCS := $(filter %_test.go, $(ALL_SRCS))
SRCS := $(filter-out %_test.go, $(ALL_SRCS))
VERSION ?= "TEST_INTERNAL"

all: PREREQUISITE build

PREREQUISITE: docs static

build: $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS) $(STATIC_DIRS)
	mkdir dist || : 
	cd $(SRC_DIRS);CGO_ENABLED=0 $(AARGS) go build -ldflags="-s -w -X main.SRATVersion=$(VERSION)" -o ../$(@)$(SUFFIX)

.PHONY: format
format:
	#docker run --rm -v $(SRC_DIRS):/code ghcr.io/swaggo/swag:latest fmt
	cd $(SRC_DIRS);swag fmt

.PHONY: clean
clean:
	go clean
	cd $(SRC_DIRS);go mod tidy
	rm -rf dist

.PHONY: test
test: $(ALL_SRCS) static
	echo $(ALL_SRCS)
	cd $(SRC_DIRS)/config;go test -cover 
	cd $(SRC_DIRS);go test -cover 

 
docs: $(SRCS)
	#docker run --rm -v $(SRC_DIRS):/code ghcr.io/swaggo/swag:latest init
	mkdir -p $(SRC_DIRS)/static && touch $(SRC_DIRS)/static/.nodelete
	cd $(SRC_DIRS);go generate

.PHONY: dev
dev: 
	cd $(SRC_DIRS);go run . -opt ../test/data/options.json -conf ../test/data/config.json $(ARGS)

$(FRONTEND_DIRS)/node_modules:
	cd $(FRONTEND_DIRS); bun install	

static: $(FRONTEND_DIRS)/node_modules
	rm -rf $(SRC_DIRS)/static
	cd $(FRONTEND_DIRS);bun run swagger
	cd $(FRONTEND_DIRS);bun run version -o $(VERSION)
	cd $(FRONTEND_DIRS);bun run build
	cp -rv $(STATIC_DIRS) $(SRC_DIRS)/static
