<!-- DOCTOC SKIP -->

# Agents

This document provides an overview of the agents and automated systems used in the SRAT project development and operation.

## Coding Agents

The SRAT project utilizes various coding agents and tools to assist with development, testing, and maintenance:

### GitHub Copilot

- **Purpose**: AI-powered code generation and assistance
- **Usage**: Integrated into VS Code for real-time code suggestions, refactoring, and documentation
- **Configuration**: See `.github/copilot-instructions.md` for project-specific guidelines
- **Languages Supported**: Go (backend), TypeScript/React (frontend)

### Pre-commit Hooks

- **Purpose**: Automated code quality checks before commits
- **Tools Included**:
  - `gosec`: Security scanning for Go code
  - Formatters: `gofmt`, `biome` for frontend
  - Linters: `govet`, `testifylint`
- **Configuration**: `.pre-commit-config.yaml`
- **Installation**: Run `pre-commit install` in the repository root

### Build Agents

- **Purpose**: Automated building and testing
- **Backend**: Uses `make` targets in `backend/Makefile` with Air for hot reload
- **Frontend**: Bun-based build system with watch mode
- **CI/CD**: Integrated with GitHub Actions for multi-arch builds (amd64, armv7, aarch64)

## Service Agents

### Backend Services

The Go backend implements several service agents for Samba administration:

- **Share Service**: Manages Samba shares configuration
- **User Service**: Handles user management and authentication
- **System Service**: Provides system information and control
- **Telemetry Service**: Optional Rollbar integration for error reporting
- **Dirty State Service**: Tracks data changes for real-time updates

### Frontend Agents

- **RTK Query**: Manages API state and caching
- **Material-UI**: Component library for consistent UI
- **SSE Client**: Handles Server-Sent Events for real-time notifications

## Development Workflow Agents

### Testing Agents

- **Backend**: `testify/suite` with `mockio/v2` for unit and integration tests
- **Frontend**: bun:test + React Testing Library for component tests
- **HTTP Testing**: `humatest` for API endpoint validation

### Documentation Agents

- **OpenAPI Generation**: Auto-generated from Go code using Huma framework
- **Markdown Validation**: `markdownlint` and link checking
- **Changelog**: Automated CHANGELOG.md updates

## Deployment Agents

### Home Assistant Integration

- **Addon Configuration**: JSON-based configuration for Home Assistant supervisor
- **Database**: Embedded SQLite with GORM ORM
- **Service Management**: Oversees Samba service lifecycle

## Security Agents

- **Dependency Scanning**: `gosec` for Go dependencies
- **Vulnerability Checks**: Integrated into pre-commit and CI pipelines
- **Access Control**: Repository permissions and branch protection

## Monitoring and Telemetry

- **Logging Agent**: Custom `tlog` package with structured logging and sensitive data masking
- **Error Reporting**: Optional Rollbar integration with user consent
- **Performance Monitoring**: Built-in profiling support (see `PPROF.md`)

For more details on specific agents, refer to the relevant documentation files in the `docs/` directory or the main `README.md`.
