TARGET_EXEC=srat
BUILD_DIR := ./dist
SRC_DIRS := ./src
FRONTEND_DIRS := ../frontend
STATIC_DIRS := ./src/static

#ALL_SRCS := $(shell find $(SRC_DIRS) -name '*.go')
ALL_SRCS := $(wildcard $(SRC_DIRS)/**/*.go)  
TEST_SRCS := $(filter %_test.go, $(ALL_SRCS))
SRCS := $(filter-out %_test.go, $(ALL_SRCS))
VERSION ?= "TEST_INTERNAL"

all: PREREQUISITE build

PREREQUISITE: docs static

build: $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS) $(STATIC_DIRS)
	mkdir dist || :  2>/dev/null
	cd $(SRC_DIRS);CGO_ENABLED=0 $(AARGS) go build -ldflags="-s -w -X main.SRATVersion=$(VERSION)" -o ../$(@)$(SUFFIX)

.PHONY: format
format:
	#docker run --rm -v $(SRC_DIRS):/code ghcr.io/swaggo/swag:latest fmt
	cd $(SRC_DIRS);go run github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc4 fmt
	cd $(SRC_DIRS);go mod tidy	
	cd $(SRC_DIRS);go run github.com/Antonboom/testifylint@v1.5.2 --fix ./...
	cd $(SRC_DIRS);go fmt

.PHONY: clean
clean:
	go clean
	rm -rf dist

.PHONY: test
test: $(ALL_SRCS) static
	cd $(SRC_DIRS);gotest -coverpkg=./... ./...

.PHONY: docs
docs: gen 

gen: $(SRCS)
	#docker run --rm -v $(SRC_DIRS):/code ghcr.io/swaggo/swag:latest init
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
