TARGET_EXEC=srat
BUILD_DIR := ./dist
SRC_DIRS := ./src

#ALL_SRCS := $(shell find $(SRC_DIRS) -name '*.go')
ALL_SRCS := $(wildcard $(SRC_DIRS)/*.go)
TEST_SRCS := $(filter %_test.go, $(ALL_SRCS))
SRCS := $(filter-out %_test.go, $(ALL_SRCS))


all: clean build

build: docs test $(BUILD_DIR)/$(TARGET_EXEC)

$(BUILD_DIR)/$(TARGET_EXEC): $(SRCS)
	mkdir dist || : 
	#go build -o $@ $^
	cd $(SRC_DIRS);go build -o ../$@ .

.PHONY: format
format:
	go mod tidy
	docker run --rm -v $(SRC_DIRS):/code ghcr.io/swaggo/swag:latest fmt

.PHONY: clean
clean:
	go clean
	rm -rf dist

.PHONY: test
test: $(ALL_SRCS)
	cd $(SRC_DIRS);go test

docs: $(SRCS)
	docker run --rm -v $(SRC_DIRS):/code ghcr.io/swaggo/swag:latest init

.PHONY: dev
dev: 
	cd $(SRC_DIRS);go run . -opt ../test/data/options.json -conf ../test/data/config.json
