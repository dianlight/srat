# Backend Development Notes


This document contains development notes, useful packages, and technical references for the SRAT backend development.

## Table of Contents


<!-- START doctoc -->
<!-- END doctoc -->

## Go Packages

### Home Assistant Integration

- **[github.com/ryanjohnsontv/go-homeassistant](https://github.com/ryanjohnsontv/go-homeassistant)** - REST and WebSocket communication with Home Assistant

### System Monitoring & Hardware

- **[github.com/shirou/gopsutil/v4](https://github.com/shirou/gopsutil)** - Cross-platform library for retrieving information on running processes and system utilization
- **[github.com/badrpc/smartctl2prom/smartctldata](https://github.com/badrpc/smartctl2prom)** - Data structures for smartctl output parsing
- **[github.com/milesbxf/smartmon-exporter/pkg/smartctl](https://github.com/milesbxf/smartmon-exporter)** - Smartctl integration utilities
- **[github.com/influxdata/telegraf/plugins/inputs/hddtemp](https://github.com/influxdata/telegraf)** - Hard drive temperature monitoring
- **[github.com/control-center/serviced/utils/iostat](https://github.com/control-center/serviced)** - I/O statistics utilities
- **[github.com/lufia/iostat](https://github.com/lufia/iostat)** - Cross-platform iostat implementation

## File System Requirements

### Minimal Size for Loopback Disks

| File System | Minimum Size | Notes                                  |
| ----------- | ------------ | -------------------------------------- |
| exFAT       | 10 MB        | Good for cross-platform compatibility  |
| FAT32       | 512 MB       | Legacy compatibility, size limitations |
| NTFS        | 10 MB        | Windows native, good compression       |

## Development Guidelines

### Code Organization

- Follow the established project structure in `/workspaces/srat/backend/src/`
- Use dependency injection with Uber FX
- Implement proper error handling with structured logging
- Use the repository pattern for data access

### API Development

- All handlers should be tested following the patterns in `setting_test.go`
- Use Huma v2 for API documentation and validation
- Implement proper CORS handling for frontend integration
- Follow RESTful conventions for endpoint design

### Configuration Management

- Use the config package for all configuration handling
- Support both JSON and environment variable configuration
- Validate configuration at startup
- Provide sensible defaults for development

## Testing Guidelines

### Test Structure

All API handler tests must follow these patterns based on the GitHub Copilot instructions:

#### Required Imports

```go
import (
    "context"
    "encoding/json"
    "net/http"
    "sync"
    "testing"

    "github.com/danielgtaylor/huma/v2/autopatch"
    "github.com/danielgtaylor/huma/v2/humatest"
    "github.com/dianlight/srat/api"
    "github.com/dianlight/srat/config"
    "github.com/dianlight/srat/converter"
    "github.com/dianlight/srat/dto"
    "github.com/dianlight/srat/service"
    "github.com/ovechkin-dm/mockio/v2/matchers"
    "github.com/ovechkin-dm/mockio/v2/mock"
    "github.com/stretchr/testify/suite"
    "go.uber.org/fx"
    "go.uber.org/fx/fxtest"
)
```

#### Test Suite Structure

- Use `testify/suite` package for structured testing
- Package naming: `package api_test`
- Suite struct naming: `{HandlerName}HandlerSuite`
- Include proper setup and teardown methods
- Test both success and error scenarios

#### Test Data Location

- Place test data in `backend/test/data/` directory
- Reference template files from `../templates/` directory
- Use meaningful test data that reflects real-world scenarios

### Mock Guidelines

- Use `mockio/v2` for mocking services and repositories
- Set up mock expectations in `SetupTest` method
- Verify mock calls in test assertions
- Clean up mocks in `TearDownTest` method

## Project Structure Rules

### Documentation Organization

- **Implementation documents** must be placed in `docs/implementation/`
- **API documentation** should be auto-generated and placed in `backend/docs/`
- **Test documentation** should be co-located with test files
- **Configuration examples** should be in `config/` or `test/data/`

### File Naming Conventions

- Test files: `*_test.go`
- Handler files: `*.go` in `api/` package
- Service files: `*.go` in `service/` package
- DTO files: `*.go` in `dto/` package
- Configuration files: `*.go` in `config/` package

### Import Organization

1. Standard library imports
2. Third-party imports
3. Project imports (grouped by package)

### Error Handling

- Use structured errors with proper HTTP status codes
- Log errors with appropriate context
- Return user-friendly error messages
- Maintain error consistency across the API

## Implementation Status

For current implementation status and architectural decisions, see:

- [Implementation Summary](../docs/implementation/IMPLEMENTATION_SUMMARY.md)
- [Persistent Notifications Implementation](../docs/implementation/IMPLEMENTATION_PERSISTENT_NOTIFICATIONS.md)
- [Removed Disks Implementation](../docs/implementation/IMPLEMENTATION_REMOVED_DISKS.md)

## Additional Resources

- [Main Project Documentation](../README.md)
- [Documentation Guidelines](../docs/DOCUMENTATION_GUIDELINES.md)
- [Home Assistant Integration](../docs/HOME_ASSISTANT_INTEGRATION.md)
- [GitHub Copilot Instructions](../.github/copilot-instructions.md)
