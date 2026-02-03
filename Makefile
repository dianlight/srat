FRONTEND_DIRS := ./frontend/
BACKEND_DIRS := ./backend/
VERSION ?= "TEST_INTERNAL_MULTIARCH"
ARCH ?= "$(shell arch)"

# Telemetry configuration - pass through environment variables
ROLLBAR_CLIENT_ACCESS_TOKEN ?= ""
ROLLBAR_ENVIRONMENT ?= ""

ALL:
	$(MAKE) -C $(BACKEND_DIRS) build AARGS="GOARCH=amd64" VERSION=$(VERSION) ARCH="x86_64" ROLLBAR_CLIENT_ACCESS_TOKEN="$(ROLLBAR_CLIENT_ACCESS_TOKEN)" ROLLBAR_ENVIRONMENT="$(ROLLBAR_ENVIRONMENT)"
	$(MAKE) -C $(BACKEND_DIRS) build AARGS="GOARCH=arm64" VERSION=$(VERSION) ARCH="aarch64" ROLLBAR_CLIENT_ACCESS_TOKEN="$(ROLLBAR_CLIENT_ACCESS_TOKEN)" ROLLBAR_ENVIRONMENT="$(ROLLBAR_ENVIRONMENT)"

.PHONY: prepare
prepare:
	pre-commit install
	pre-commit install --hook-type post-commit
	pre-commit install --hook-type pre-push
	$(MAKE) -C $(BACKEND_DIRS) PREREQUISITE
	cd $(FRONTEND_DIRS); bun install

.PHONY: docs
docs:  docs-fix docs-toc docs-validate

.PHONY: docs-toc
docs-toc:
	@echo "Generating Table of Contents for Markdown files..."
	@if command -v bun >/dev/null 2>&1; then \
		echo "Using bun package manager for doctoc"; \
		bun install -g doctoc; \
	elif command -v npm >/dev/null 2>&1; then \
		echo "Using npm package manager for doctoc"; \
		npm install -g doctoc; \
	else \
		echo "Error: No package manager found. Please install bun or npm first."; \
		exit 1; \
	fi
	@find . -name "*.md" -not -path "./node_modules/*" -not -path "./frontend/node_modules/*" -not -path "./backend/src/vendor/*" | xargs bunx doctoc --github

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
	bunx markdownlint-cli2 "**/*.md" "#frontend/node_modules" "#backend/src/vendor" --fix

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
	if command -v lychee >/dev/null 2>&1; then \
		echo "✅ Lychee found (link checker)"; \
	else \
		echo "⚠️  Lychee not found (optional but recommended)"; \
		echo "    Install: https://lychee.cli.rs/#/installation"; \
	fi; \
	if command -v vale >/dev/null 2>&1; then \
		echo "✅ Vale found (prose linter)"; \
	else \
		echo "⚠️  Vale not found (optional but recommended)"; \
		echo "    Install: https://vale.sh/docs/vale-cli/installation/"; \
	fi; \
	echo "✅ Documentation validation dependencies check complete"; \
	echo "JavaScript runtime: $$NODE_RUNTIME, Package manager: $$PACKAGE_MANAGER"

.PHONY: docs-help
docs-help:
	@echo "📚 SRAT Documentation Commands (GitHub Flavored Markdown)"
	@echo "=========================================================="
	@echo ""
	@echo "Available make targets:"
	@echo "  docs-check     - Check if all dependencies are installed"
	@echo "  docs-install   - Install documentation validation tools"
	@echo "  docs-validate  - Run all documentation validation checks"
	@echo "  docs-fix       - Auto-fix common documentation formatting issues"
	@echo "  docs-toc       - Generate table of contents for markdown files"
	@echo "  docs-help      - Show this help message"
	@echo ""
	@echo "Direct script usage:"
	@echo "  ./scripts/validate-docs.sh        - Run full validation"
	@echo "  ./scripts/validate-docs.sh --fix  - Auto-fix formatting"
	@echo "  ./scripts/validate-docs.sh --help - Show script help"
	@echo ""
	@echo "Validation tools:"
	@echo "  - markdownlint-cli2: Markdown syntax and formatting (GFM)"
	@echo "  - Lychee: Link and image validation"
	@echo "  - cspell: Spell checking"
	@echo "  - Vale: Prose linting and style checking (GFM)"
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
	@echo "Note: Lychee and Vale should be installed separately (see docs-check)"
	@if command -v bun >/dev/null 2>&1; then \
		echo "Using bun to install JS-based documentation tools..."; \
		bun add -g markdownlint-cli2 cspell prettier doctoc; \
	elif command -v npm >/dev/null 2>&1; then \
		echo "Using npm to install JS-based documentation tools..."; \
		npm install -g markdownlint-cli2 cspell prettier doctoc; \
	else \
		echo "Error: Neither bun nor npm found. Please install one of them first."; \
		exit 1; \
	fi
	@echo ""
	@echo "To install optional tools:"
	@echo "  Lychee: https://lychee.cli.rs/#/installation"
	@echo "  Vale: https://vale.sh/docs/vale-cli/installation/"

.PHONY: clean
clean:
	cd $(FRONTEND_DIRS); bun clean
	$(MAKE) -C $(BACKEND_DIRS) clean

.PHONY: gemini
gemini:
	bun --bun $(shell which gemini)

.PHONY: check
check: docs-check security
	pre-commit run --files $(find . -name "*.md" | grep -v "node_modules")

.PHONY: security
security:
	$(MAKE) -C $(BACKEND_DIRS) gosec

# -----------------------------------------------------------------------------
# Git housekeeping helpers
# -----------------------------------------------------------------------------
.PHONY: clean-local-branches clean-local-branches-dry clean-local-tags clean-local-tags-dry

# Remove local branches that do not exist on remote and have no commits in the last 2 weeks
clean-local-branches:
	bash scripts/cleanup-old-local-branches.sh

# Dry-run: show which local branches would be removed
clean-local-branches-dry:
	bash scripts/cleanup-old-local-branches.sh --dry

# Remove local tags that do not exist on remote and point to commits older than 2 weeks
clean-local-tags:
	bash scripts/cleanup-old-local-tags.sh

# Dry-run: show which local tags would be removed
clean-local-tags-dry:
	bash scripts/cleanup-old-local-tags.sh --dry
