FRONTEND_DIRS := ./frontend/
BACKEND_DIRS := ./backend/


ALL: 
	cd $(BACKEND_DIRS);$(MAKE) 

PREREQUISITE:
	cd $(BACKEND_DIRS);$(MAKE) PREREQUISITE	

.PHONY: clean
clean:
	cd $(FRONTEND_DIRS); bun clean
	cd $(BACKEND_DIRS);$(MAKE) clean

