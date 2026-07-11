# [FEATURE]: Zeroconf mDNS Registration from Addon (Lab)

**Target Repo:** `srat`  
**Status:** 📅 Planned  
**Issue Link:** [Optional]

## 🎯 Objective
Implement zeroconf/mDNS service registration directly from the SRAT addon (Go backend) so that SMB shares are discoverable on the network via mDNS/Bonjour even when the Home Assistant custom component is not installed. This is a Lab feature gated behind `experimental_lab_mode` and is mutually exclusive with the custom_component's equivalent mDNS registration solution.

## 🛠️ Technical Specifications
- **Inputs:** Addon configuration (SMB server name, port, enabled interfaces), Lab mode flag
- **Outputs:** mDNS service registration for `_smb._tcp` on port 445 (SMB), visible in network browsers (Finder, Windows Explorer, etc.)
- **Dependencies:** 
  - `github.com/grandcat/zeroconf` Go library
  - Go 1.26 backend
  - SRAT addon lifecycle (start/stop)
  - Frontend settings for Lab mode toggle
- **Mutual Exclusivity:** When this addon-based mDNS is enabled, the custom_component's mDNS registration must be disabled (and vice versa). This should be enforced via configuration validation.

## 📝 Task List
- [ ] Task 1: Add `github.com/grandcat/zeroconf` dependency to backend go.mod
- [ ] Task 2: Create mDNS service registration module in backend (e.g., `backend/src/services/zeroconf/`)
- [ ] Task 3: Implement interface filtering logic (exclude loopback, docker, veth, hassio, br- interfaces)
- [ ] Task 4: Add mDNS service start/stop lifecycle tied to addon startup/shutdown
- [ ] Task 5: Add configuration options for mDNS (instance name, port, enabled interfaces, TXT records)
- [ ] Task 6: Add Lab feature gating (`experimental_lab_mode`) for mDNS settings in frontend
- [ ] Task 7: Add mutual exclusivity validation with custom_component mDNS
- [ ] Task 8: Unit testing for interface filtering and service registration logic
- [ ] Task 9: Integration testing with addon lifecycle
- [ ] Task 10: Update CHANGELOG.md with feature entry
- [ ] Task 11: Update documentation (README, settings docs) explaining Lab feature and mutual exclusivity
- [ ] Task 12: Code review and cleanup
- [ ] Task 13: Final testing and validation
- [ ] Task 14: Capture lessons learned and update documentation
- [ ] Task 15: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

### Code Example (from user)
```go
package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/grandcat/zeroconf"
)

func getPhysicalInterfaces() []net.Interface {
	var valid []net.Interface
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Impossibile enumerare le interfacce: %v", err)
		return nil
	}

	for _, iface := range ifaces {
		name := iface.Name
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "hassio") ||
			strings.HasPrefix(name, "br-") {
			continue
		}
		valid = append(valid, iface)
	}
	return valid
}

func main() {
	port := 445
	txt := []string{}

	ifaces := getPhysicalInterfaces()
	if len(ifaces) == 0 {
		log.Println("Nessuna interfaccia fisica trovata, uso tutte come fallback")
		ifaces = nil
	}

	server, err := zeroconf.Register(
		"MioAddonSamba", // instance name
		"_smb._tcp",
		"local.",
		port,
		txt,
		ifaces,
	)
	if err != nil {
		log.Fatalf("Errore registrazione mDNS SMB: %v", err)
	}
	defer server.Shutdown()

	log.Printf("Servizio SMB pubblicato via mDNS sulla porta %d", port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Deregistro il servizio mDNS SMB...")
}
```

### Key Implementation Details
1. **Service Registration**: Use `zeroconf.Register()` with:
   - Instance name: configurable (default: addon hostname or "SRAT-Samba")
   - Service type: `_smb._tcp`
   - Domain: `local.`
   - Port: 445 (standard SMB)
   - TXT records: optional, can include `u=` for username, `p=` for path, etc.
   - Interfaces: filtered physical interfaces only

2. **Interface Filtering**: Exclude:
   - Loopback (`net.FlagLoopback`)
   - Down interfaces (`net.FlagUp == 0`)
   - Docker/veth/hassio/br- prefixed interfaces

3. **Lifecycle Integration**: 
   - Start mDNS registration on addon startup (after SMB server is ready)
   - Stop gracefully on addon shutdown (SIGTERM/SIGINT)
   - Handle errors gracefully (log but don't crash addon)

4. **Lab Feature Gating** (Frontend):
   - Follow `HomeAssistantPanel.tsx` pattern:
     ```tsx
     const experimentalLabMode = Boolean(watch("experimental_lab_mode"));
     const labLabel = (text: string) => (
       <> {text} <ScienceOutlinedIcon color="warning" fontSize="small" /> </>
     );
     {experimentalLabMode && (
       <Feature label={labLabel("mDNS Registration")} />
     )}
     ```

5. **Mutual Exclusivity**:
   - Add validation in both addon config and custom_component config
   - If addon mDNS enabled, custom_component should skip mDNS registration
   - Document clearly in both places

6. **Configuration Schema** (addon config.yaml):
   ```yaml
   mdns:
     enabled: false
     instance_name: "SRAT-Samba"
     port: 445
     txt_records: []
     interfaces: [] # empty = auto-detect
   ```

## 🔗 Code References & TODOs
- Backend service location: `backend/src/services/zeroconf/` (to be created)
- Frontend settings: `frontend/src/pages/settings/` (Lab mode section)
- Custom component: `custom_components/srat/` (mutual exclusivity check)
- Addon config: `backend/src/config/` or `addon/config.yaml`
- Existing Lab feature pattern: `frontend/src/pages/settings/HomeAssistantPanel.tsx`