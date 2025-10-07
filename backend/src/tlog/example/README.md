# TLog Package Examples

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Files](#files)
- [Running the Examples](#running-the-examples)
  - [Main Demonstration Program](#main-demonstration-program)
  - [Example Tests](#example-tests)
  - [Logger Demo Functions](#logger-demo-functions)
- [Building](#building)
- [Dependencies](#dependencies)
- [Features Demonstrated](#features-demonstrated)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

This directory contains examples demonstrating how to use the `tlog` package.

## Files

- **`main.go`** - Complete demonstration program showing all tlog features
- **`logger_demo.go`** - Example functions demonstrating Logger struct usage
- **`examples_test.go`** - Go example tests that can be run with `go test`

## Running the Examples

### Main Demonstration Program

```bash
go run main.go
```

This will run a comprehensive demonstration of all tlog features including:

- Basic logging functions
- Level management
- Error handling
- Context logging
- Performance optimization
- Custom logger instances
- Event callbacks
- Callback management and error handling

### Example Tests

```bash
go test -run Example
```

This will run the example test functions that demonstrate:

- Creating new loggers
- Using different log levels
- Context-aware logging methods
- Callback functionality
- Embedded slog functionality

### Logger Demo Functions

The `logger_demo.go` file contains the `ExampleLogger()` function that can be called from other programs to demonstrate basic logger usage.

## Building

To build the example program:

```bash
go build -o tlog-example main.go logger_demo.go
./tlog-example
```

## Dependencies

The examples require the parent `tlog` package. The `go.mod` file includes a replace directive to use the local development version.

## Features Demonstrated

1. **Basic Logging** - All log levels (Trace, Debug, Info, Notice, Warn, Error)
2. **Level Management** - Setting and checking log levels
3. **Context Logging** - Using context-aware logging methods
4. **Performance** - Level checking to avoid expensive operations
5. **Custom Loggers** - Creating logger instances with specific levels
6. **Callbacks** - Registering and managing event callbacks
7. **Error Handling** - Proper error handling and callback recovery
8. **slog Integration** - Using embedded slog.Logger functionality
