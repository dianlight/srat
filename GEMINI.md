# Gemini Project Configuration

This file provides project-specific guidance to the Gemini agent.

## Project Overview

This project appears to be a web application with a Go backend and a JavaScript/TypeScript frontend.

- **Backend:** Located in the `backend/` directory, written in Go and using huma, gorm and uber/fx for dependency management.
- **Frontend:** Located in the `frontend/` directory, likely using a modern JavaScript framework and using React and MUI.


## Development Environment

- The backend uses Go modules for dependency management (`go.mod`).
- The frontend uses `bun` for package management (`bun.lockb`, `package.json`).

## Commit structure

Follow the Conventional Commits format strictly for commit messages. 
Use the structure below:

<type>[optional scope]: <gitmoji> <description>

[optional footer]


Guidelines:
  1. **Type and Scope**: Choose an appropriate type (e.g., `feat`, `fix`) and optional scope to describe the affected module or feature.
  2. **Gitmoji**: Include a relevant `gitmoji` that best represents the nature of the change.
  3. **Description**: Write a concise, informative description in the header; use backticks if referencing code or specific terms.
  4. **Body**: For additional details, use a well-structured body section:
     - Use bullet points (`*`) for clarity.
     - Clearly describe the motivation, context, or technical details behind the change, if applicable.
     
Commit messages should be clear, informative, and professional, aiding readability and project tracking."

## Build & Test Commands

- **Root:**
  - To build all architectures: `make`
  - To prepare the development environment (install pre-commit hooks, backend prerequisites, and frontend dependencies): `make prepare`
  - To clean build artifacts: `make clean`
  - To run the Gemini CLI: `make gemini`
- **Backend:**
  - To generate code: `make -C backend gen`
  - To build: `make -C backend build`
  - To test: `make -C backend test`
  - To lint: `make -C backend format`
- **Frontend:**
  - To install dependencies: `cd frontend && bun install`
  - To generate code: `cd frontend && bun gen`
  - To build: `cd frontend && bun build`
  - To test: `cd frontend && bun test`
  - To lint: `cd frontend && bun lint`


## Architectural Notes

- The software is designed to run in a Homeassistant addon environment.
- The backend exposes an OpenAPI specification at `backend/docs/openapi.yaml`.
- The frontend use a generator for generate OpenAPI client
- Backend e frontend comunication are made by REST services or SSE
- Only DTO struct are used in REST and SSE comunications
- In the backend the conversion between non dto struct and dto are made by coverter package that is genereated by goverter
- In the frontend MUI 7.x is used and Grid are used as Grid2.
- `backend/docs/openapi.yaml` is generated from code by code generation
- `frontend/src/store/sratApi.ts` is generated from `backend/docs/openapi.yaml` by code generation

## Dependencies

- **Backend:** See `backend/go.mod` for Go dependencies.
- **Frontend:** See `frontend/package.json` for Node.js dependencies.
