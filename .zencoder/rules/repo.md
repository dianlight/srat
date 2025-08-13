---
description: Repository Information Overview
alwaysApply: true
---

# Repository Information Overview

> **IMPORTANT**: Always check `.github/copilot-*.md` files for project-specific rules and guidelines. Rules in Copilot rule files have precedence over other documentation.

## Repository Summary
SRAT (SambaNAS REST Administration Tool) is a system designed to provide a simplified user interface for configuring SAMBA. It's developed to work within Home Assistant as an addon but can also be used in other contexts. The project consists of a React-based frontend and a Go-based backend with a RESTful API.

## Repository Structure
- **backend/**: Go-based backend with RESTful API implementation
- **frontend/**: React-based frontend with TypeScript
- **config/**: Home Assistant configuration files
- **docs/**: Project documentation
- **scripts/**: Utility scripts for development and deployment

### Main Repository Components
- **Backend API Server**: Go-based REST API for SAMBA configuration
- **Frontend Web UI**: React application for user interface
- **Home Assistant Integration**: Configuration for Home Assistant addon

## Projects

### Backend (Go API Server)
**Configuration File**: backend/src/go.mod

#### Language & Runtime
**Language**: Go
**Version**: 1.24.3
**Build System**: Make
**Package Manager**: Go Modules

#### Dependencies
**Main Dependencies**:
- github.com/gorilla/mux v1.8.1 (HTTP router)
- github.com/glebarez/sqlite v1.11.0 (SQLite database)
- gorm.io/gorm v1.30.1 (ORM)
- github.com/danielgtaylor/huma/v2 v2.34.1 (API framework)
- github.com/rollbar/rollbar-go v1.4.8 (Error reporting)

#### Build & Installation
```bash
cd backend && make build
```

#### Testing
**Framework**: Go testing package
**Test Location**: backend/src/**/*_test.go
**Run Command**:
```bash
cd backend/src && go test ./...
```

### Frontend (React Web UI)
**Configuration File**: frontend/package.json

#### Language & Runtime
**Language**: TypeScript/JavaScript
**Version**: TypeScript 5.8.3
**Build System**: Bun
**Package Manager**: Bun (v1.2.20)

#### Dependencies
**Main Dependencies**:
- react v19.1.0
- @mui/material v7.1.1 (UI components)
- @reduxjs/toolkit v2.8.2 (State management)
- react-router-dom v7.6.2 (Routing)
- react-hook-form v7.57.0 (Form handling)
- rollbar v2.26.4 (Error reporting)

**Development Dependencies**:
- @biomejs/biome v2.1.4 (Linting)
- @types/react v19.1.9
- bun-html-live-reload v1.0.4

#### Build & Installation
```bash
cd frontend && bun install && bun run build
```

## Database
SRAT uses SQLite for persistence via the GORM ORM. The backend initializes the database with resilience-focused defaults including WAL journal mode, busy timeout settings, and foreign key constraints. The database path can be set via the `--db` flag when running the server.

## Development Environment
**Prerequisites**:
- Bun (JavaScript runtime and package manager)
- Go (for backend development)
- pre-commit (for git hooks)

**Setup Commands**:
```bash
# Install pre-commit hooks and dependencies
make prepare

# Install documentation validation tools
make docs-install

# Build all components
make ALL
```

## Testing & Validation
**Backend Testing**: Go's built-in testing framework
**Documentation Validation**: Comprehensive validation tools including markdown linting, link checking, and spell checking
**Security Scanning**: Uses gosec to scan Go code for security issues

**Validation Commands**:
```bash
# Check all documentation
make docs-validate

# Run security checks
make security
```

## Project Guidelines
**Copilot Rules**: The repository contains detailed development guidelines in:
- `.github/copilot-rules.md`: Contains comprehensive coding standards for Go, documentation rules, and project-specific conventions
- `.github/copilot-instructions.md`: Provides detailed instructions for test creation, package documentation, and security practices

These files should be consulted for authoritative guidance on project conventions and have precedence over other documentation.
