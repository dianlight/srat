# PPROF USE DOCUMENTATION

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents**

- [Build with pprof](#build-with-pprof)
  - [CPU USE PPROF](#cpu-use-pprof)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Build with pprof

Using build tags pprof the srat-server also expose standard pprof endpoints

Refer to https://pkg.go.dev/net/http/pprof

### CPU USE PPROF

Use pprof tool to see cpu profiling

```bash
go tool -C ./src/ pprof -http=":" ${HOMEASSISTANT_IP}:3000/debug/pprof/profile?seconds=10

```
