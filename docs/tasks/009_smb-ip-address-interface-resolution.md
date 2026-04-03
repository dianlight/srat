# [FIX]: Use IP Addresses Instead of Interface Names in SMB.Conf

**Target Repo:** `srat`  **Status:** ✅ Complete  **Issue Link:** [srat#356](https://github.com/dianlight/srat/issues/356)

## 🎯 Objective

Replace network interface name references in the Samba `interfaces =` directive with explicit IP addresses to honour IPv4 preference settings. Using interface names (e.g. `enp0s25`) can cause Samba to bind to IPv6 addresses when both are present on the interface, leading to connectivity issues. Resolving interface names to IP addresses at config-generation time ensures the correct address family is used.

> _Context for Copilot: The Samba config is generated via `backend/src/templates/smb.gtpl` and populated by `server_process_service.go`. The current template emits interface names as-is. Network interface information is available via `gopsutil/v4/net` or `net.Interfaces()`. The `AppConfig` carries an `interfaces` field (slice of interface name strings)._

## 🛠️ Technical Specifications

- **Inputs:**
  - `AppConfig.Interfaces []string` — configured interface names (e.g. `["lo", "enp0s25"]`)
  - `AppConfig.DisableIPv6 bool` — when true, only emit IPv4 addresses
  - Live network interface data (from `net.Interfaces()` or `gopsutil`)

- **Outputs:**
  - `interfaces = 127.0.0.1 192.168.1.10 ...` in `smb.conf` (IP addresses, not names)
  - Fallback: if an interface cannot be resolved (e.g. not up), log a warning and skip it
  - When `DisableIPv6` is true, exclude all IPv6 addresses from the list

- **Dependencies:**
  - `backend/src/service/server_process_service.go` — Samba config template data preparation
  - `backend/src/templates/smb.gtpl` — `interfaces =` and `bind interfaces only =` directives
  - `backend/src/dto/app_config.go` — `AppConfig` struct
  - `golang.org/x/sys` or `net` stdlib — interface address resolution

## 📝 Task List

- [x] Task 1: Add a helper function `resolveInterfaceIPs(names []string, ipv6 bool) ([]string, error)` in a suitable package (e.g. `service/` or `internal/netutil/`) that maps each interface name to its associated IP addresses
- [x] Task 2: Call `resolveInterfaceIPs` in `server_process_service.go` when building the Samba config template data, replacing the raw interface name list with resolved IP strings
- [x] Task 3: Update `smb.gtpl` to iterate over the resolved IP list (no change if template already uses a generic `range .Interfaces`)
- [x] Task 4: When `AppConfig.DisableIPv6 == true`, filter out any IPv6 addresses from the resolved list (via CLI flag `--ipv4-only`)
- [x] Task 5: Log a warning (using `slog/tlog`) for each interface name that cannot be resolved or has no addresses assigned
- [x] Task 6: Unit tests — `resolveInterfaceIPs` with mocked interface list: IPv4-only, IPv6-only, dual-stack, missing interface
- [x] Task 7: Integration / template rendering test — verify generated `smb.conf` contains IP addresses, not interface names
- [x] Task 8: Confirm that the new behavior is reflected in documentation if the `interfaces` config field description mentions behavior change (e.g. update README or config docs)
- [x] Task 9: Review code for clarity, maintainability, and adherence to project conventions (e.g. error handling, logging)
- [x] Task 10: Run `go build ./...` and `go vet ./...` to confirm no regressions
- [x] Task 11: Run existing tests: `cd backend/src && go test ./...` — all must pass
- [x] Task 12: Run `go_diagnostics` (gopls) on modified files and fix any reported issues
- [x] Task 13: Update documentation if `interfaces` config field description mentions behavior change
- [x] Task 14: Update `CHANGELOG.md` with a note about the change in how `interfaces` are specified in `smb.conf`

## 🧠 Implementation Notes (Copilot Context)

### Revised Architecture (CLI flag-based)

The `DisableIPv6` control is now a CLI flag `--ipv4-only` (default false). The flag value flows:
`main-server.go` → `staticConfig.DisableIPv6` (dto.ContextState) → `ServerService.state` → `CreateSambaConfigStream()`

### Resolving IPs from interface names

```go
import "net"

func resolveInterfaceIPs(names []string, allowIPv6 bool) ([]string, error) {
    var ips []string
    for _, name := range names {
        iface, err := net.InterfaceByName(name)
        if err != nil {
            // log warning: interface not found
            continue
        }
        addrs, err := iface.Addrs()
        if err != nil {
            continue
        }
        for _, addr := range addrs {
            var ip net.IP
            switch v := addr.(type) {
            case *net.IPNet:
                ip = v.IP
            case *net.IPAddr:
                ip = v.IP
            }
            if ip == nil || ip.IsLinkLocalUnicast() {
                continue
            }
            if !allowIPv6 && ip.To4() == nil {
                continue // skip IPv6
            }
            ips = append(ips, ip.String())
        }
    }
    return ips, nil
}
```

### SMB.conf `interfaces` directive

The Samba `interfaces` option accepts both IP addresses and CIDR notation. Using bare IPs (`127.0.0.1`) is the simplest and most portable form. `lo` should always be included as `127.0.0.1`.

### Loopback handling

Always include `127.0.0.1` regardless of whether `lo` is in the configured interface list, as Samba requires localhost for internal IPC.

## 🔗 Code References & TODOs

- [ ] [srat#356](https://github.com/dianlight/srat/issues/356) — "Use IP Address not interface name in SMB.Conf to honor IPv4 preference"
- [ ] `backend/src/cmd/srat-server/main-server.go` — add `--ipv4-only` flag
- [ ] `backend/src/dto/context.go` — add `DisableIPv6 bool` to ContextState
- [ ] `backend/src/service/server_process_service.go` — add `resolveInterfaceIPs` helper and integrate into `CreateSambaConfigStream`
- [ ] `backend/src/templates/smb.gtpl` — verify generic `range .interfaces` usage (likely no change needed)
- [ ] `backend/src/service/server_process_service_test.go` — update tests for IP address output and IPv6 filtering

## 📋 Implementation Plan (Agreed 2026-03-23)

1. **Add CLI flag** in `main-server.go`: `noIPv6 *bool` with default false, pass to `staticConfig.DisableIPv6`
2. **Extend ContextState** in `dto/context.go`: add `DisableIPv6 bool` field
3. **Create helper** `resolveInterfaceIPs(names []string, allowIPv6 bool) ([]string, error)` in `server_process_service.go`
4. **Modify `CreateSambaConfigStream()`**: resolve interfaces before `ConfigToMap()`, always include `127.0.0.1`
5. **Update tests**: unit test helper, adjust integration test expectations
6. **Validate**: build, vet, test; verify `smb.conf` contains IPs
7. **Documentation**: update task notes and CHANGELOG if needed
