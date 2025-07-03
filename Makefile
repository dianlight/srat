FRONTEND_DIRS := ./frontend/
BACKEND_DIRS := ./backend/
COMPONENTS_DIRS := ./custom_components/
VERSION ?= "TEST_INTERNAL_MULTIARCH"
ARCH ?= "$(shell arch)"
#SUFFIX ?= "_$(ARCH)"



ALL:
	$(MAKE) -C $(BACKEND_DIRS) AARGS="GOARCH=amd64" VERSION=$(VERSION) ARCH="x86_64"
	$(MAKE) -C $(BACKEND_DIRS) build AARGS="GOARM=7 GOARCH=arm" VERSION=$(VERSION) ARCH="armv7"
	$(MAKE) -C $(BACKEND_DIRS) build AARGS="GOARCH=arm64"  VERSION=$(VERSION) ARCH="aarch64"

.PHONY: prepare
prepare:
	# Backend
	pre-commit install
	pre-commit install --hook-type post-commit
	$(MAKE) -C $(BACKEND_DIRS) PREREQUISITE
	# Frontend
	cd $(FRONTEND_DIRS); bun install; cd ..
	# Components
	python3 -m pip install --requirement requirements.txt



.PHONY: clean
clean:
	cd $(FRONTEND_DIRS); bun clean
	$(MAKE) -C $(BACKEND_DIRS) clean


.PHONY: lint
lint:
	$(MAKE) -C $(BACKEND_DIRS) lint
	cd $(FRONTEND_DIRS); bun run lint
	cd ..
	cd $(COMPONENTS_DIRS); ruff format .; ruff check . --fix

.PHONY: gen
gen:
	$(MAKE) -C $(BACKEND_DIRS) gen
	cd $(FRONTEND_DIRS); bun run gen
	cd ..
	openapi-python-client generate  --path ./backend/docs/openapi.json --output-path ./custom_components/srat_companion/api

.PHONY: dev_ha
.ONESHELL:
dev_ha:
	# Create config dir if not present
	if [[ ! -d "${PWD}/config" ]]; then
		mkdir -p "${PWD}/config"
		hass --config "${PWD}/config" --script ensure_config
	fi

	# Set the path to custom_components
	## This let's us have the structure we want <root>/custom_components/srat_companion
	## while at the same time have Home Assistant configuration inside <root>/config
	## without resulting to symlinks.
	export PYTHONPATH="${PYTHONPATH}:${PWD}/custom_components"

	# Start Home Assistant
	hass --config "${PWD}/config" --debug

.PHONY: dev_remote
dev_remote:
	umount -l /mnt/remote_comp > /dev/null 2>&1 || :
	mkdir -p /mnt/remote_comp > /dev/null 2>&1 || :
	sshfs -o reconnect,ServerAliveInterval=15,ServerAliveCountMax=3 root@192.168.0.68:/homeassistant/custom_components /mnt/remote_comp
	cp -rv custom_components/srat_companion /mnt/remote_comp/
	umount -l /mnt/remote_comp > /dev/null 2>&1 || :
.PHONY: gemini
gemini:
	bun --bun $(shell which gemini)
	
.PHONY: lint
lint:
	$(MAKE) -C $(BACKEND_DIRS) lint
	cd $(FRONTEND_DIRS); bun run lint
	cd $(COMPONENTS_DIRS); ruff format .; ruff check . --fix

.PHONY: gen
gen:
	$(MAKE) -C $(BACKEND_DIRS) gen
	cd $(FRONTEND_DIRS); bun run gen
	openapi-python-client generate --config ./custom_components/srat_companion/openapi-client.yaml --path ./backend/docs/openapi.json --output ./custom_components/srat_companion/api

.PHONY: dev_ha
.ONESHELL: 
dev_ha:
	# Create config dir if not present
	if [[ ! -d "${PWD}/config" ]]; then
		mkdir -p "${PWD}/config"
		hass --config "${PWD}/config" --script ensure_config
	fi

	# Set the path to custom_components
	## This let's us have the structure we want <root>/custom_components/srat_companion
	## while at the same time have Home Assistant configuration inside <root>/config
	## without resulting to symlinks.
	export PYTHONPATH="${PYTHONPATH}:${PWD}/custom_components"

	# Start Home Assistant
	hass --config "${PWD}/config" --debug
	
