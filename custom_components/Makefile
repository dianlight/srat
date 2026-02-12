COMPONENT_DIR := srat
PYTHON_FILES := $(COMPONENT_DIR)/*.py

.PHONY: help
help:
	@echo "SRAT Custom Component – Development Targets"
	@echo "============================================="
	@echo ""
	@echo "  make install    Install development dependencies"
	@echo "  make format     Auto-format code with ruff"
	@echo "  make lint       Lint code with ruff"
	@echo "  make typecheck  Run mypy static type checking"
	@echo "  make check      Run all checks (format-check + lint + typecheck)"
	@echo "  make fix        Auto-fix linting issues where possible"
	@echo "  make clean      Remove build/cache artifacts"

.PHONY: install
install:
	pip install -r requirements_dev.txt

.PHONY: format
format:
	ruff format $(COMPONENT_DIR)

.PHONY: format-check
format-check:
	ruff format --check $(COMPONENT_DIR)

.PHONY: lint
lint:
	ruff check $(COMPONENT_DIR)

.PHONY: fix
fix:
	ruff check --fix $(COMPONENT_DIR)
	ruff format $(COMPONENT_DIR)

.PHONY: typecheck
typecheck:
	mypy $(COMPONENT_DIR)

.PHONY: check
check: format-check lint typecheck
	@echo ""
	@echo "✓ All checks passed"

.PHONY: clean
clean:
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name .mypy_cache -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name .ruff_cache -exec rm -rf {} + 2>/dev/null || true
