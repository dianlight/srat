ALL: frontend/node_modules PREREQUISITE
	cd backend;$(MAKE)

PREREQUISITE:
	cd backend;$(MAKE) docs
	cd frontend; bun swagger; bun run build

.PHONY: clean
clean:
	cd frontend; bun clean
	cd backend;$(MAKE) clean

frontend/node_modules:
	cd frontend; bun install