SHELL := /bin/bash
GO := $(or $(shell command -v go 2>/dev/null), go)
GOCMD := $(GO)
GOTEST := $(GOCMD) test
GOBUILD := $(GOCMD) build
GOMOD := $(GOCMD) mod

PKGS := $(shell $(GOCMD) list ./...)
BIN := smartmontools-go

# Where go install will place binaries. Prefer GOBIN, fall back to GOPATH/bin
GOBIN := $(shell $(GOCMD) env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell $(GOCMD) env GOPATH)/bin
endif

# staticcheck binary path (either on PATH or in GOBIN)
STATICCHECK := $(or $(shell command -v staticcheck 2>/dev/null),$(GOBIN)/staticcheck)

# gomarkdoc binary path (either on PATH or in GOBIN)
GOMARKDOC := $(or $(shell command -v gomarkdoc 2>/dev/null),$(GOBIN)/gomarkdoc)

.PHONY: help build test coverage fmt vet lint tidy mod-download run-example release prerelease apidoc clean

help:
	@echo "Makefile for smartmontools-go"
	@echo ""
	@echo "Available targets:"
	@echo "  test           Run unit tests for all packages"
	@echo "  coverage       Run tests and show coverage summary"
	@echo "  fmt            Run gofmt on the project"
	@echo "  vet            Run go vet on the project"
	@echo "  lint           Run staticcheck if available"
	@echo "  tidy           Run go mod tidy"
	@echo "  mod-download   Download modules (go mod download)"
	@echo "  run-example    Run the example in examples/basic"
	@echo "  apidoc         Generate API documentation (APIDOC.md)"
	@echo "  release        Create and push a release tag (usage: make release [VERSION_TYPE=major|minor|patch])"
	@echo "  prerelease     Create and push a prerelease tag (usage: make prerelease [VERSION_TYPE=major|minor|patch])"
	@echo "  clean          Remove build artifacts"

test:
	@echo "Running tests..."
	$(GOTEST) ./...

coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.txt ./... -covermode=atomic || true
	@echo "Summary:"
	@if [ -f coverage.txt ]; then $(GOCMD) tool cover -func=coverage.txt | tail -n 1; fi

fmt:
	@echo "Formatting code (gofmt)..."
	@gofmt -s -w .

.PHONY: fmt-check
fmt-check:
	@echo "Checking code format (gofmt)..."
	@! test -n "$(shell gofmt -l .)" || (echo "gofmt needs to be applied" && exit 1)

vet:
	@echo "Running go vet..."
	@$(GOCMD) vet ./...

lint:
	@echo "Running staticcheck if available..."
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; else echo "staticcheck not installed; skip (install: go install honnef.co/go/tools/cmd/staticcheck@latest)"; fi

.PHONY: ci-lint
ci-lint:
	@echo "Running CI linting: fmt-check, vet, staticcheck"
	@$(MAKE) fmt-check
	@$(MAKE) vet
	@if ! command -v staticcheck >/dev/null 2>&1; then \
		echo "staticcheck not found — installing to $(GOBIN)..."; \
		$(GOCMD) install honnef.co/go/tools/cmd/staticcheck@latest; \
		if [ ! -x "$(STATICCHECK)" ]; then echo "installation failed or $(STATICCHECK) not found"; exit 1; fi; \
	fi
	@echo "Running staticcheck..."
	@"$(STATICCHECK)" ./...

.PHONY: ci
ci: tidy mod-download ci-lint test
	@echo "CI: all checks passed"

tidy:
	@echo "Running go mod tidy..."
	@$(GOMOD) tidy

mod-download:
	@echo "Downloading modules..."
	@$(GOMOD) download

run-example:
	@echo "Running examples/basic..."
	@cd examples/basic && $(GOCMD) run ./

apidoc:
	@echo "Generating API documentation..."
	@if command -v gomarkdoc >/dev/null 2>&1 || [ -x "$(GOMARKDOC)" ]; then "$(GOMARKDOC)" -o APIDOC.md .; else echo "gomarkdoc not installed; install: go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest"; exit 1; fi

release:
	@VERSION_TYPE=$(or $(VERSION_TYPE), patch); \
	CURRENT_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	CURRENT_VERSION=$$(echo $$CURRENT_TAG | sed 's/^v//' | sed 's/-.*//'); \
	MAJOR=$$(echo $$CURRENT_VERSION | cut -d. -f1); \
	MINOR=$$(echo $$CURRENT_VERSION | cut -d. -f2); \
	PATCH=$$(echo $$CURRENT_VERSION | cut -d. -f3); \
	case $$VERSION_TYPE in \
		major) NEW_MAJOR=$$((MAJOR + 1)); NEW_MINOR=0; NEW_PATCH=0 ;; \
		minor) NEW_MAJOR=$$MAJOR; NEW_MINOR=$$((MINOR + 1)); NEW_PATCH=0 ;; \
		patch) NEW_MAJOR=$$MAJOR; NEW_MINOR=$$MINOR; NEW_PATCH=$$((PATCH + 1)) ;; \
		*) echo "Invalid VERSION_TYPE: $$VERSION_TYPE"; exit 1 ;; \
	esac; \
	NEW_TAG=v$$NEW_MAJOR.$$NEW_MINOR.$$NEW_PATCH; \
	echo "Creating release tag $$NEW_TAG"; \
	git tag $$NEW_TAG && git push origin $$NEW_TAG

prerelease:
	@VERSION_TYPE=$(or $(VERSION_TYPE), patch); \
	CURRENT_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	HAS_PRERELEASE=$$(echo $$CURRENT_TAG | grep -c '-'); \
	if [ $$HAS_PRERELEASE -eq 1 ]; then \
		PRERELEASE_PART=$$(echo $$CURRENT_TAG | sed 's/.*-//'); \
		N=$$(echo $$PRERELEASE_PART | cut -d. -f2); \
		NEW_N=$$((N + 1)); \
		NEW_TAG=$$(echo $$CURRENT_TAG | sed "s/\.[0-9]*$$/\.$$NEW_N/"); \
	else \
		CURRENT_VERSION=$$(echo $$CURRENT_TAG | sed 's/^v//' | sed 's/-.*//'); \
		MAJOR=$$(echo $$CURRENT_VERSION | cut -d. -f1); \
		MINOR=$$(echo $$CURRENT_VERSION | cut -d. -f2); \
		PATCH=$$(echo $$CURRENT_VERSION | cut -d. -f3); \
		case $$VERSION_TYPE in \
			major) NEW_MAJOR=$$((MAJOR + 1)); NEW_MINOR=0; NEW_PATCH=0 ;; \
			minor) NEW_MAJOR=$$MAJOR; NEW_MINOR=$$((MINOR + 1)); NEW_PATCH=0 ;; \
			patch) NEW_MAJOR=$$MAJOR; NEW_MINOR=$$MINOR; NEW_PATCH=$$((PATCH + 1)) ;; \
			*) echo "Invalid VERSION_TYPE: $$VERSION_TYPE"; exit 1 ;; \
		esac; \
		NEW_TAG=v$$NEW_MAJOR.$$NEW_MINOR.$$NEW_PATCH-beta.0; \
	fi; \
	echo "Creating prerelease tag $$NEW_TAG"; \
	git tag $$NEW_TAG && git push origin $$NEW_TAG

clean:
	@echo "Cleaning..."
	@rm -f $(BIN) coverage.out APIDOC.md
