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

.PHONY: docs-validate
docs-validate:
	@echo "Running documentation validation..."
	@if command -v bun >/dev/null 2>&1; then \
		echo "Using bun package manager for validation tools"; \
	elif command -v npm >/dev/null 2>&1; then \
		echo "Using npm package manager for validation tools"; \
	else \
		echo "Warning: No package manager found. Some validation tools may fail."; \
	fi
	@./scripts/validate-docs.sh

.PHONY: docs-fix
docs-fix:
	@echo "Auto-fixing documentation formatting..."
	@if command -v bun >/dev/null 2>&1; then \
		echo "Using bun package manager for formatting tools"; \
	elif command -v npm >/dev/null 2>&1; then \
		echo "Using npm package manager for formatting tools"; \
	else \
		echo "Warning: No package manager found. Auto-fix may fail."; \
	fi
	@./scripts/validate-docs.sh --fix

.PHONY: docs-check
docs-check:
	@echo "Checking documentation validation dependencies..."
	@NODE_RUNTIME=""; \
	PACKAGE_MANAGER=""; \
	if command -v node >/dev/null 2>&1; then \
		echo "✅ Node.js found"; \
		NODE_RUNTIME="node"; \
	elif command -v bun >/dev/null 2>&1; then \
		echo "✅ bun found (can be used as Node.js alternative)"; \
		NODE_RUNTIME="bun"; \
	else \
		echo "❌ No JavaScript runtime found (Node.js or bun required)"; \
		echo "Please install Node.js or bun first"; \
		exit 1; \
	fi; \
	if command -v bun >/dev/null 2>&1; then \
		echo "✅ bun package manager found"; \
		PACKAGE_MANAGER="bun"; \
	elif command -v npm >/dev/null 2>&1; then \
		echo "✅ npm package manager found"; \
		PACKAGE_MANAGER="npm"; \
	else \
		echo "❌ No package manager found (bun or npm required)"; \
		exit 1; \
	fi; \
	if command -v pre-commit >/dev/null 2>&1; then \
		echo "✅ pre-commit found"; \
	else \
		echo "⚠️  pre-commit not found (optional for git hooks)"; \
	fi; \
	echo "✅ Documentation validation dependencies check complete"; \
	echo "JavaScript runtime: $$NODE_RUNTIME, Package manager: $$PACKAGE_MANAGER"

.PHONY: docs-help
docs-help:
	@echo "📚 SRAT Documentation Commands"
	@echo "=============================="
	@echo ""
	@echo "Available make targets:"
	@echo "  docs-check     - Check if all dependencies are installed"
	@echo "  docs-install   - Install documentation validation tools"
	@echo "  docs-validate  - Run all documentation validation checks"
	@echo "  docs-fix       - Auto-fix common documentation formatting issues"
	@echo "  docs-help      - Show this help message"
	@echo ""
	@echo "Direct script usage:"
	@echo "  ./scripts/validate-docs.sh        - Run full validation"
	@echo "  ./scripts/validate-docs.sh --fix  - Auto-fix formatting"
	@echo "  ./scripts/validate-docs.sh --help - Show script help"
	@echo ""
	@echo "Package manager support:"
	@if command -v bun >/dev/null 2>&1; then \
		echo "  ✅ bun detected (runtime + package manager)"; \
	elif command -v npm >/dev/null 2>&1 && command -v node >/dev/null 2>&1; then \
		echo "  ✅ Node.js + npm detected"; \
	elif command -v node >/dev/null 2>&1; then \
		echo "  ⚠️  Node.js found but no package manager"; \
	else \
		echo "  ❌ No runtime or package manager found"; \
	fi
	@echo ""
	@echo "For more information, see docs/DOCUMENTATION_GUIDELINES.md"

.PHONY: docs-install
docs-install:
	@echo "Installing documentation validation tools..."
	@if command -v bun >/dev/null 2>&1; then \
		echo "Using bun to install documentation tools..."; \
		bun add -g markdownlint-cli2 markdown-link-check cspell prettier; \
	elif command -v npm >/dev/null 2>&1; then \
		echo "Using npm to install documentation tools..."; \
		npm install -g markdownlint-cli2 markdown-link-check cspell prettier; \
	else \
		echo "Error: Neither bun nor npm found. Please install one of them first."; \
		exit 1; \
	fi

.PHONY: clean
clean:
	cd $(FRONTEND_DIRS); bun clean
	$(MAKE) -C $(BACKEND_DIRS) clean

.PHONY: gemini
gemini:
	bun --bun $(shell which gemini)

.PHONY: check
check: docs-check
	pre-commit run --all-files