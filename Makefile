FRONTEND_DIRS := ./frontend/
BACKEND_DIRS := ./backend/


ALL: $(FRONTEND_DIRS)/node_modules PREREQUISITE
	cd $(BACKEND_DIRS);$(MAKE)

PREREQUISITE:
	cd $(BACKEND_DIRS);$(MAKE) docs
	cd $(FRONTEND_DIRS); bun swagger; bun run build

.PHONY: clean
clean:
	cd $(FRONTEND_DIRS); bun clean
	cd $(BACKEND_DIRS);$(MAKE) clean

