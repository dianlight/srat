# Gemini Project Configuration

This file provides project-specific guidance to the Gemini agent.

## Project Overview

This project appears to be a web application with a Go backend and a JavaScript/TypeScript frontend.

- **Backend:** Located in the `backend/` directory, written in Go and using huma, gorm and uber/fx for dependency management.
- **Frontend:** Located in the `frontend/` directory, likely using a modern JavaScript framework and using React and MUI.


## Development Environment

- The backend uses Go modules for dependency management (`go.mod`).
- The frontend uses `bun` for package management (`bun.lockb`, `package.json`).

## Build & Test Commands

*This section is a placeholder and should be filled in with the correct commands.*

- **Backend:**
  - To generate code: `make -C backend gen`
  - To build: `make -C backend build`
  - To test: `make -C backend test`
- **Frontend:**
  - To install dependencies: `cd frontend && bun install`
  - To generate code: `cd frontend && bun gen`
  - To build: `cd frontend && bun build`
  - To test: `cd frontend && bun test`

## Architectural Notes

- The software is designed to run in a Homeassistant addon environment.
- The backend exposes an OpenAPI specification at `backend/docs/openapi.yaml`.
- The frontend use a generator for generate OpenAPI client
- Backend e frontend comunication are made by REST services or SSE
- Only DTO struct are used in REST and SSE comunications
- In the backend the conversion between non dto struct and dto are made by coverter package that is genereated by goverter
- In the frontend MUI 7.x is used and Grid are used as Grid2

## Dependencies

- **Backend:** See `backend/go.mod` for Go dependencies.
- **Frontend:** See `frontend/package.json` for Node.js dependencies.
