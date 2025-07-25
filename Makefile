FRONTEND_DIRS := ./frontend/
BACKEND_DIRS := ./backend/
VERSION ?= "TEST_INTERNAL_MULTIARCH"
ARCH ?= "$(shell arch)"
#SUFFIX ?= "_$(ARCH)"



ALL:
	$(MAKE) -C $(BACKEND_DIRS) AARGS="GOARCH=amd64" VERSION=$(VERSION) ARCH="x86_64"
	$(MAKE) -C $(BACKEND_DIRS) build AARGS="GOARM=7 GOARCH=arm" VERSION=$(VERSION) ARCH="armv7"
	$(MAKE) -C $(BACKEND_DIRS) build AARGS="GOARCH=arm64"  VERSION=$(VERSION) ARCH="aarch64"

.PHONY: prepare
prepare:
	pre-commit install
	pre-commit install --hook-type post-commit
	$(MAKE) -C $(BACKEND_DIRS) PREREQUISITE
	cd $(FRONTEND_DIRS); bun install


.PHONY: clean
clean:
	cd $(FRONTEND_DIRS); bun clean
	$(MAKE) -C $(BACKEND_DIRS) clean

.PHONY: gemini
gemini:
	bun --bun $(shell which gemini)