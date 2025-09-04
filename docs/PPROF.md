# PPROF USE DOCUMENTATION

## Build with pprof

Using build tags pprof the srat-server also expose standard pprof endpoints

Refer to https://pkg.go.dev/net/http/pprof

### CPU USE PPROF

Use pprof tool to see cpu profiling

```bash
go tool -C ./src/ pprof -http=":" ${HOMEASSISTANT_IP}:3000/debug/pprof/profile?seconds=10

```
