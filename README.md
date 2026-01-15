# SambaNAS Rest Administration Tool ![SRAT](https://github.com/dianlight/srat/raw/main/docs/full_logo.png)

[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/dianlight/srat?include_prereleases)](https://img.shields.io/github/v/release/dianlight/srat?include_prereleases)
[![GitHub last commit](https://img.shields.io/github/last-commit/dianlight/srat)](https://img.shields.io/github/last-commit/dianlight/srat)
[![GitHub issues](https://img.shields.io/github/issues-raw/dianlight/srat)](https://img.shields.io/github/issues-raw/dianlight/srat)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/dianlight/srat)](https://img.shields.io/github/issues-pr/dianlight/srat)
[![GitHub](https://img.shields.io/github/license/dianlight/srat)](https://img.shields.io/github/license/dianlight/srat)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit)](https://github.com/pre-commit/pre-commit)
[![codecov](https://codecov.io/github/dianlight/srat/graph/badge.svg?token=6C61IZRDEJ)](https://codecov.io/github/dianlight/srat)

[![Backend Unit Test Coverage](https://img.shields.io/badge/Backend_Unit_Tests-43.4%25-yellow?logo=go)](docs/TEST_COVERAGE.md "Backend Go unit test coverage")
[![Frontend Unit Test Coverage](https://img.shields.io/badge/Frontend_Unit_Tests-69.51%25-green?logo=typescript)](docs/TEST_COVERAGE.md "Frontend TypeScript unit test coverage")
[![Global Unit Test Coverage](https://img.shields.io/badge/Global_Unit_Tests-53.8%25-yellow)](docs/TEST_COVERAGE.md "Overall unit test coverage (weighted average)")

SRAT (SambaNAS REST Administration Tool) is a new system designed to provide a simplified user interface for configuring SAMBA. It has been developed to work within Home Assistant, specifically for this addon, but can also be used in other contexts.

Currently under development and in an alpha state, SRAT is set to become the preferred system for configuring and using this addon, eventually "retiring" the YAML configuration.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Installation](#installation)
  - [Local Development](#local-development)
- [Usage](#usage)
  - [Feature Documentation](#feature-documentation)
- [Database](#database)
- [Sponsor](#sponsor)
- [Known Issues](#known-issues)
- [Enhanced Logging System](#enhanced-logging-system)
  - [Key Logging Features](#key-logging-features)
  - [Quick Logging Examples](#quick-logging-examples)
- [Documentation Validation](#documentation-validation)
- [Security scanning](#security-scanning)
- [Building the Project](#building-the-project)
- [Testing and Coverage](#testing-and-coverage)
  - [Running Tests](#running-tests)
  - [Coverage Metrics](#coverage-metrics)
  - [Coverage Goals](#coverage-goals)
- [Contribute](#contribute)
- [License](#license)
- [Development: pre-commit hooks](#development-pre-commit-hooks)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

:construction_worker: This is a part for new SambaNas2 Home Assistant Addon. :construction_worker:

## Installation

The installation of this add-on is straightforward and similar to any other Home Assistant add-on.

[Add our Home Assistant add-ons repository][repository] to your Home Assistant instance:

[![Add-on repository badge](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fdianlight%2Fhassio-addons)

or

[Add our Home Assistant BETA add-ons repository][beta-repository] to your Home Assistant instance:

[![Beta repository badge](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fdianlight%2Fhassio-addons-beta)

[repository]: https://github.com/dianlight/hassio-addons
[beta-repository]: https://github.com/dianlight/hassio-addons-beta

### Local Development

For local development setup, follow the instructions below for backend and frontend development.

## Usage

SRAT can be used to manage Samba shares, users, and configurations via a modern web UI or REST API. For detailed feature usage, see the documentation in the `docs/` folder or access the API docs at `/docs` when running the backend server.

### Feature Documentation

- [Settings Documentation](docs/SETTINGS_DOCUMENTATION.md) - Complete reference for all SRAT settings
- [SMB over QUIC](docs/SMB_OVER_QUIC.md) - Enhanced performance and security with QUIC transport protocol
- [Telemetry Configuration](docs/TELEMETRY_CONFIGURATION.md) - Configure error reporting and monitoring
- [Home Assistant Integration](docs/HOME_ASSISTANT_INTEGRATION.md) - Integration with Home Assistant

## Database

SRAT uses SQLite for persistence via the GORM ORM. The backend initializes the database with resilience-focused defaults:

- journal_mode=WAL for safe readers during writes
- busy_timeout=5000ms to reduce transient SQLITE_BUSY
- synchronous=NORMAL tuned for WAL
- foreign_keys=ON
- cache=shared
- connection pool limited to 1 open/idle connection (embedded DB best practice)

You can set the database path via the `--db` flag when running the server or CLI. For example:

- File on disk (recommended for production): `--db /data/srat.db`
- In-memory for tests/dev: `--db "file::memory:?cache=shared&_pragma=foreign_keys(1)"`

## Sponsor

<a href="https://github.com/sponsors/dianlight"><img src="https://img.shields.io/github/sponsors/dianlight?style=flat-square&logo=githubsponsors&logoColor=%23EA4AAA" alt="Github Sponsor"></a>
<a href="https://www.buymeacoffee.com/ypKZ2I0"><img src="https://img.buymeacoffee.com/button-api/?text=Buy me a coffee&emoji=&slug=ypKZ2I0" alt="Buy Me a Coffee"/></a>

## Known Issues

## Enhanced Logging System

SRAT includes a comprehensive logging system with the `tlog` package. For detailed information about logging capabilities, see [TLOG Repository](https://github.com/dianlight/tlog).

### Key Logging Features

- **Professional Formatting**: Powered by `samber/slog-formatter` with automatic error structuring
- **Security-First**: Automatic masking of sensitive data (passwords, tokens, API keys, IP addresses)
- **Developer-Friendly**: Color-coded output with terminal detection and level-based coloring
- **Production-Ready**: Thread-safe operations with configurable output formats
- **Context-Aware**: Automatic extraction and display of request/trace/user context

### Quick Logging Examples

```go
import "github.com/dianlight/srat/tlog"

// Basic usage with enhanced formatting
tlog.Info("Server started", "port", 8080, "version", "1.0.0")
tlog.Error("Database connection failed", "error", err, "host", "localhost")

// Context-aware logging
ctx := context.WithValue(context.Background(), "request_id", "req-12345")
tlog.InfoContext(ctx, "Processing request", "method", "GET", "path", "/api/users")

// Enable security features
tlog.EnableSensitiveDataHiding(true) // Auto-masks passwords, tokens, IPs
tlog.EnableColors(true)              // Color output (auto-disabled if not terminal)
```

## Documentation Validation

SRAT includes comprehensive documentation validation tools with **GitHub Flavored Markdown (GFM)** support:

```shell
# Check all documentation (GFM-aware)
make docs-validate

# Auto-fix formatting issues
make docs-fix

# Show all documentation commands
make docs-help
```

The validation includes:

- **Markdown linting and formatting** (markdownlint with GFM support)
- **Link and image checking** (Lychee - fast and reliable)
- **Spell checking** (cspell with project vocabulary)
- **Prose linting** (Vale for style and consistency)
- **Content structure validation**

For more details, see [Documentation Validation Setup](docs/DOCUMENTATION_VALIDATION_SETUP.md) and [Documentation Guidelines](docs/DOCUMENTATION_GUIDELINES.md).

## Security scanning

This project uses gosec to scan Go code for common security issues.

Quick usage:

- Run full repo security check: `make security`
- Or only backend: `cd backend && make gosec`

Notes:

- Generated code is excluded with `-exclude-generated`.
- The Makefile will install gosec automatically if it's missing (via `go install`).

## Building the Project

```shell
# Build backend
cd backend && make build

# Build frontend
cd frontend && bun run build

# Build all architectures
make ALL
```

## Testing and Coverage

SRAT maintains high test coverage across both backend and frontend. The coverage badges at the top of this README are automatically updated.

📊 **[View Detailed Coverage Report & History →](docs/TEST_COVERAGE.md)**

### Running Tests

```shell
# Run backend tests with individual package coverage
cd backend && make test

# Run frontend tests with coverage
cd frontend && bun test --coverage

# Update coverage badges in README and TEST_COVERAGE.md
bash scripts/update-coverage-badges.sh
```

### Coverage Metrics

The project tracks three coverage metrics:

- **Backend Coverage**: Go package coverage (individual package basis)
- **Frontend Coverage**: TypeScript/React component coverage (line coverage)
- **Global Coverage**: Weighted average (60% backend, 40% frontend)

Coverage badge colors:

- 🟢 Green (≥80%): Excellent
- 🟢 Light Green (≥60%): Good
- 🟡 Yellow (≥40%): Acceptable
- 🟠 Orange (≥20%): Needs improvement
- 🔴 Red (<20%): Critical

### Coverage Goals

- Minimum backend package coverage: 2%
- Minimum frontend function coverage: 90%
- Target global coverage: 60%+

## Contribute

You can use this section to highlight how people can contribute to your project.

You can add information on how they can open issues or how they can sponsor the project.

## License

<!-- [(Back to top)](#table-of-contents) -->

[Apache 2.0 license](./LICENSE)

## Development: pre-commit hooks

This repository manages all git hooks via pre-commit. Don’t add scripts under .git/hooks or set core.hooksPath.

Quick start:

- Install pre-commit (pipx, pip, brew, apk add py3-pip + pip install pre-commit)
- Install hooks: pre-commit install && pre-commit install --hook-type pre-push
- Run all hooks: pre-commit run --all-files

Enforced hooks:

- On commit: gosec security scan for backend Go changes (high severity/high confidence)
- On push: backend quick build + test

See .pre-commit-config.yaml for full list. The legacy .githooks directory is deprecated.
