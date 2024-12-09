TARGET_EXEC=srat
BUILD_DIR := ./dist
SRC_DIRS := ./src
FRONTEND_DIRS := ../frontend/
STATIC_DIRS := $(FRONTEND_DIRS)/out/

#ALL_SRCS := $(shell find $(SRC_DIRS) -name '*.go')
ALL_SRCS := $(wildcard $(SRC_DIRS)/**/*.go)  
TEST_SRCS := $(filter %_test.go, $(ALL_SRCS))
SRCS := $(filter-out %_test.go, $(ALL_SRCS))


all: clean docs static build

build: $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS) $(STATIC_DIRS)
	mkdir dist || : 
	#go build -o $@ $^
	cd $(SRC_DIRS);go build -o ../$@ .

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
	cd $(SRC_DIRS)/lsblk;go test -cover 
	cd $(SRC_DIRS);go test -cover 

 
docs: $(SRCS)
	#docker run --rm -v $(SRC_DIRS):/code ghcr.io/swaggo/swag:latest init
	cd $(SRC_DIRS);swag init --pd

.PHONY: dev
dev: 
	cd $(SRC_DIRS);go run . -opt ../test/data/options.json -conf ../test/data/config.json

$(FRONTEND_DIRS)/node_modules:
	cd $(FRONTEND_DIRS); bun install	

static: $(STATIC_DIRS) $(FRONTEND_DIRS)/node_modules
	rm -rf $(SRC_DIRS)/static
	cd $(FRONTEND_DIRS);bun run build
	cp -rv $(STATIC_DIRS) $(SRC_DIRS)/static
	#cd $(FRONTEND_DIRS);bun run version
