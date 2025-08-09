# SambaNAS Rest Administration Tool ![SRAT](https://github.com/dianlight/srat/raw/main/docs/full_logo.png)

[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/dianlight/srat?include_prereleases)](https://img.shields.io/github/v/release/dianlight/srat?include_prereleases)
[![GitHub last commit](https://img.shields.io/github/last-commit/dianlight/srat)](https://img.shields.io/github/last-commit/dianlight/srat)
[![GitHub issues](https://img.shields.io/github/issues-raw/dianlight/srat)](https://img.shields.io/github/issues-raw/dianlight/srat)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/dianlight/srat)](https://img.shields.io/github/issues-pr/dianlight/srat)
[![GitHub](https://img.shields.io/github/license/dianlight/srat)](https://img.shields.io/github/license/dianlight/srat)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit)](https://github.com/pre-commit/pre-commit)

SRAT (SambaNAS REST Administration Tool) is a new system designed to provide a simplified user interface for configuring SAMBA. It has been developed to work within Home Assistant, specifically for this addon, but can also be used in other contexts.

Currently under development and in an alpha state, SRAT is set to become the preferred system for configuring and using this addon, eventually "retiring" the YAML configuration.

## Key Features

- **Modern Web UI**: React-based frontend with TypeScript support
- **RESTful API**: Go-based backend with comprehensive API documentation
- **Enhanced Logging**: Advanced logging system with the `tlog` package featuring:
  - Custom log levels (TRACE, DEBUG, INFO, NOTICE, WARN, ERROR, FATAL)
  - Automatic color support with terminal detection
  - Sensitive data protection (passwords, tokens, API keys automatically masked)
  - Enhanced error formatting with tree-structured stack traces
  - Context-aware logging with request tracking
  - Thread-safe operations and asynchronous callback system
- **Docker Integration**: Seamless integration with Home Assistant addons
- **Real-time Updates**: Server-sent events for live configuration updates

:construction_worker: This is a part for new SambaNas2 Home Assistant Addon. :construction_worker:

## Installation

Use my addon SmabaNAS2

The installation of this add-on is pretty straightforward and not different in
comparison to installing any other Hass.io add-on.

[Add our Hass.io add-ons repository][repository] to your Hass.io instance.

[![Open your Home Assistant instance and show the add add-on repository dialog with a specific repository URL pre-filled.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fdianlight%2Fhassio-addons)

or

[Add our Hass.io BETA add-ons repository][beta-repository] to your Hass.io instance.

[![Open your Home Assistant instance and show the add add-on repository dialog with a specific repository URL pre-filled.](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2Fdianlight%2Fhassio-addons-beta)

[repository]: https://github.com/dianlight/hassio-addons
[beta-repository]: https://github.com/dianlight/hassio-addons-beta

## Sponsor

<a href="https://github.com/sponsors/dianlight"><img src="https://img.shields.io/github/sponsors/dianlight?style=flat-square&logo=githubsponsors&logoColor=%23EA4AAA&link=https%3A%2F%2Fgithub.com%2Fsponsors%2Fdianlight" alt="Github Sponsor"></a>

<a href="https://www.buymeacoffee.com/ypKZ2I0"><img src="https://img.buymeacoffee.com/button-api/?text=Buy me a coffee&emoji=&slug=ypKZ2I0&button_colour=FFDD00&font_colour=000000&font_family=Cookie&outline_colour=000000&coffee_colour=ffffff" alt="Buy Me a Coffee"/></a>

<!--
# Quick Start Demo

![Demo Preview](https://picsum.photos/1920/1080)

I believe that you should bring value to the reader as soon as possible. You should be able to get the user up and running with your project with minimal friction.

If you have a quickstart guide, this is where it should be.

Alternatively, you can add a demo to show what your project can do.

# Table of Contents

This is a table of contents for your project. It helps the reader navigate through the README quickly.
- [Project Title](#project-title)
- [Quick Start Demo](#quick-start-demo)
- [Table of Contents](#table-of-contents)
- [Installation](#installation)
- [Usage](#usage)
- [Development](#development)
- [Contribute](#contribute)
- [License](#license)

# Installation
[(Back to top)](#table-of-contents)

> **Note**: For longer README files, I usually add a "Back to top" buttton as shown above. It makes it easy to navigate.

This is where your installation instructions go.

You can add snippets here that your readers can copy-paste with click:

```shell
```shell
gh repo clone navendu-pottekkat/awesome-readme
```

# Usage
[(Back to top)](#table-of-contents)

Next, you have to explain how to use your project. You can create subsections under here to explain more clearly.

# Development
[(Back to top)](#table-of-contents)

For developers who want to contribute to SRAT, here are the setup instructions:

## Prerequisites
- Node.js OR bun (JavaScript runtime - bun can replace Node.js)
- bun or npm (package manager)
- pre-commit (for git hooks)
- Go (for backend development)

**Note**: bun can serve as both JavaScript runtime and package manager, making it a complete Node.js replacement for this project.

## Setup Development Environment

```shell
# Clone the repository
git clone https://github.com/dianlight/srat.git
cd srat

# Check documentation dependencies
make docs-check

# Install pre-commit hooks and dependencies
make prepare

# Install documentation validation tools
make docs-install
```

## Enhanced Logging System

SRAT includes a comprehensive logging system with the `tlog` package. For detailed information about logging capabilities, see [backend/src/tlog/README.md](./backend/src/tlog/README.md).

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

SRAT includes comprehensive documentation validation tools:

```shell
# Check all documentation
make docs-validate

# Auto-fix formatting issues
make docs-fix

# Show all documentation commands
make docs-help
```

The validation includes:
- Markdown linting and formatting
- Link checking
- Spell checking
- Content structure validation
- Security scanning

## Building the Project

```shell
# Build backend
cd backend && make build

# Build frontend
cd frontend && bun run build

# Build all architectures
make ALL
```

# Contribute
[(Back to top)](#table-of-contents)

You can use this section to highlight how people can contribute to your project.

You can add information on how they can open issues or how they can sponsor the project.

-->

## License

<!-- [(Back to top)](#table-of-contents) -->

[Apache 2.0 license](./LICENSE)
