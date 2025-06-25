# Gemini Project Configuration

This file provides project-specific guidance to the Gemini agent.

## Project Overview

This project appears to be a web application with a Go backend and a JavaScript/TypeScript frontend.

- **Backend:** Located in the `backend/` directory, written in Go.
- **Frontend:** Located in the `frontend/` directory, likely using a modern JavaScript framework.

## Development Environment

- The backend uses Go modules for dependency management (`go.mod`).
- The frontend uses `bun` for package management (`bun.lockb`, `package.json`).

## Build & Test Commands

*This section is a placeholder and should be filled in with the correct commands.*

- **Backend:**
  - To build: `make -C backend build`
  - To test: `make -C backend test`
- **Frontend:**
  - To build: `cd frontend && bun install && bun build`
  - To test: `cd frontend && bun test`

## Architectural Notes

- The backend exposes an OpenAPI specification at `backend/docs/openapi.yaml`.
# - The application seems to have a concept of "blueprints" for automation and scripts, located in `config/blueprints`.

## Dependencies

- **Backend:** See `backend/go.mod` for Go dependencies.
- **Frontend:** See `frontend/package.json` for Node.js dependencies.
