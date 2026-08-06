# [FEATURE]: Zeroconf mDNS Registration from Addon (Lab)

**Target Repo:** `srat`  
**Status:** ✅ Complete  
**Issue Link:** [Optional]

## 🎯 Objective
Implement zeroconf/mDNS service registration directly from the SRAT addon (Go backend) so that SMB shares are discoverable on the network via mDNS/Bonjour even when the Home Assistant custom component is not installed. This is a Lab feature gated behind `experimental_lab_mode` and is mutually exclusive with the custom_component's equivalent mDNS registration solution.

## 🛠️ Technical Specifications
- **Inputs:** Hostname (from Samba settings), Lab mode flag, optional interface whitelist
- **Outputs:** mDNS service registration for `_smb._tcp` on port 445 (SMB), visible in network browsers (Finder, Windows Explorer, etc.)
- **Dependencies:**
  - `github.com/grandcat/zeroconf` Go library
  - Go 1.26 backend
  - SRAT addon lifecycle (start/stop)
  - Frontend settings for Lab mode toggle
- **Mutual Exclusivity:** When this addon-based mDNS is enabled, the custom_component's mDNS registration must be disabled (and vice versa). This is enforced via `ValidateSettings` in the backend.

## 📝 Task List
- [x] Task 1: Add `github.com/grandcat/zeroconf` dependency to backend go.mod
- [x] Task 2: Extend `MDNSService` in `backend/src/service/mdns_service.go` to call `zeroconf.Register`
- [x] Task 3: Implement interface filtering logic (exclude loopback, docker, veth, hassio, br- interfaces)
- [x] Task 4: Add mDNS service start/stop lifecycle tied to settings changes and fx OnStop
- [x] Task 5: Persist only `addon_mdns_registration` and `addon_mdns_interfaces`; derive instance name, port, and TXT from existing settings
- [x] Task 6: Add Lab feature gating (`experimental_lab_mode`) for mDNS settings in frontend
- [x] Task 7: Add mutual exclusivity validation with custom_component mDNS
- [x] Task 8: Unit testing for interface filtering and service registration logic
- [x] Task 9: Integration testing with addon lifecycle
- [x] Task 10: Update CHANGELOG.md with feature entry
- [x] Task 11: Update task documentation explaining Lab feature and mutual exclusivity
- [ ] Task 12: Code review and cleanup (deferred to PR)
- [ ] Task 13: Final testing and validation (deferred to PR)
- [ ] Task 14: Capture lessons learned and update documentation (deferred to PR)
- [ ] Task 15: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

### Final Design
- Only two new persisted settings were added:
  - `Settings.AddonMDNSRegistration *bool`
  - `Settings.AddonMDNSInterfaces []string`
- The mDNS instance name is derived from `Settings.Hostname` using the same
  NetBIOS sanitization as the Samba config: uppercase, truncate to 15 chars,
  replace non-alphanumeric characters with `-`.
- Service details are fixed: `_smb._tcp`, domain `local.`, port `445`, TXT
  record `path=/`.
- `SystemCapabilities.AvailableMDNSInterfaces` exposes the list of eligible
  interfaces to the frontend so the user can pick from a dropdown.

### Code References
- Backend direct mDNS logic: `backend/src/service/mdns_service.go`
- Backend validation: `backend/src/service/setting_service.go` (`ValidateSettings`)
- Backend capabilities: `backend/src/api/system.go` (`GetCapabilitiesHandler`)
- Frontend lab-gated controls: `frontend/src/pages/settings/panels/HomeAssistantPanel.tsx`
- Frontend tests: `frontend/src/pages/settings/panels/__tests__/HomeAssistantPanel.mdns.test.tsx`
- Backend tests: `backend/src/service/mdns_service_test.go`, `backend/src/service/setting_service_test.go`
- DTO changes: `backend/src/dto/settings.go`, `backend/src/dto/system_capabilities.go`
- OpenAPI/codegen artifacts: `backend/docs/openapi.*`, `frontend/src/store/sratApi.ts`