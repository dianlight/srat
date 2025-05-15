FRONTEND_DIRS := ./frontend/
BACKEND_DIRS := ./backend/
VERSION ?= "TEST_INTERNAL_MULTIARCH"
ARCH ?= "$(shell arch)"
SUFFIX ?= "_$(ARCH)"



ALL:
#ifeq ($(ARCH),"amd64")
	cd $(BACKEND_DIRS);$(MAKE) AARGS="GOARCH=amd64" SUFFIX="_x86_64" _SUFFIX="$(SUFFIX)" VERSION=$(VERSION)
#else ifeq ($(ARCH), "armv7")
	cd $(BACKEND_DIRS);$(MAKE) AARGS="GOARM=7 GOARCH=arm" SUFFIX="_armv7" _SUFFIX="$(SUFFIX)" VERSION=$(VERSION)
#else ifeq ($(ARCH), "aarch64")
	cd $(BACKEND_DIRS);$(MAKE) AARGS="GOARCH=arm64" SUFFIX="_aarch64" _SUFFIX="$(SUFFIX)" VERSION=$(VERSION)
#else
#	$(info "Unknown architecture")
#endif

#BUILD:
#	cd $(BACKEND_DIRS);$(MAKE) AARGS="$(AARGS)" SUFFIX="$(SUFFIX)" VERSION=$(VERSION)


#PREREQUISITE:
#	cd $(BACKEND_DIRS);$(MAKE) PREREQUISITE

.PHONY: prepare
prepare:
	pre-commit install
	cd $(BACKEND_DIRS);$(MAKE) PREREQUISITE
	cd $(FRONTEND_DIRS); bun install


.PHONY: clean
clean:
	cd $(FRONTEND_DIRS); bun clean
	cd $(BACKEND_DIRS);$(MAKE) clean
