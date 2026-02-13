COMPONENT_DIR := srat
TEST_DIR := tests
PYTHON_FILES := $(COMPONENT_DIR)/*.py

.PHONY: help
help:
	@echo "SRAT Custom Component – Development Targets"
	@echo "============================================="
	@echo ""
	@echo "  make install          Install dev deps (auto-detects Alpine apk vs pip)"
	@echo "  make install-pip      Install dev deps via pip only"
	@echo "  make format           Auto-format code with ruff"
	@echo "  make format-check     Check formatting without modifying files"
	@echo "  make lint             Lint code with ruff"
	@echo "  make typecheck        Run mypy static type checking"
	@echo "  make test             Run component tests with pytest"
	@echo "  make test-ci          Run tests with coverage (XML for Codecov)"
	@echo "  make check            Run all checks (format-check + lint + typecheck + test)"
	@echo "  make fix              Auto-fix linting issues + reformat"
	@echo "  make clean            Remove build/cache artifacts"
	@echo "  make mount_remote     Mount remote HA custom_components via sshfs"
	@echo "  make install_remote   Rsync component to remote HA instance"

.PHONY: install
install:
	@if command -v apk >/dev/null 2>&1; then \
		echo "Alpine detected – installing via apk..."; \
		apk add --no-cache python3 py3-pip py3-mypy py3-ruff py3-pytest; \
	else \
		echo "Installing via pip..."; \
		pip install -r requirements_dev.txt; \
	fi

.PHONY: install-pip
install-pip:
	pip install -r requirements_dev.txt

.PHONY: format
format:
	ruff format $(COMPONENT_DIR) $(TEST_DIR)

.PHONY: format-check
format-check:
	ruff format --check $(COMPONENT_DIR) $(TEST_DIR)

.PHONY: lint
lint:
	ruff check $(COMPONENT_DIR) $(TEST_DIR)

.PHONY: fix
fix:
	ruff check --fix $(COMPONENT_DIR) $(TEST_DIR)
	ruff format $(COMPONENT_DIR) $(TEST_DIR)

.PHONY: typecheck
typecheck:
	mypy $(COMPONENT_DIR)

.PHONY: test
test:
	cd .. && python -m pytest custom_components/tests -v -c custom_components/pyproject.toml

.PHONY: test-ci
test-ci:
	cd .. && python -m pytest custom_components/tests -v -c custom_components/pyproject.toml \
		--cov=custom_components/srat --cov-report=xml:custom_components/coverage.xml

.PHONY: check
check: format-check lint typecheck test
	@echo ""
	@echo "✓ All checks passed"

.PHONY: clean
clean:
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name .mypy_cache -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name .ruff_cache -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name .pytest_cache -exec rm -rf {} + 2>/dev/null || true

.PHONY: mount_remote
mount_remote:
	@umount -l /mnt/remote2 > /dev/null 2>&1 || :
	@mkdir -p /mnt/remote2 > /dev/null 2>&1 || :
	@sshfs -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3 ${HOMEASSISTANT_IP}:/config/custom_components /mnt/remote2

.PHONY: install_remote
install_remote: /mnt/remote2/srat/manifest.json
	@rsync -rvuzP --del  ./srat ${HOMEASSISTANT_IP}:/config/custom_components