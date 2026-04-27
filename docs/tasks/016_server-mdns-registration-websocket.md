# [FEATURE]: Server mDNS Registration via WebSocket to Custom Component

**Target Repo:** `srat` 
**Status:** 📅 Planned 
**Issue Link:** _TBD_

## 🎯 Objective

Enable automatic mDNS (Bonjour/Avahi) registration of the Samba server when the Home Assistant custom component establishes a WebSocket connection. The server hostname will be announced via python-zeroconf to allow Home Assistant and network clients to discover the Samba server. The feature must gracefully handle component disconnection by removing the mDNS announcement after a 30-second timeout.

## 🛠️ Technical Specifications

- **Inputs:**
  - WebSocket `helo` handshake from custom component (already stored in `dto.ContextState.HAWsComponent`)
  - Server process event `ServerProcess/CLEAN` from eventBus
  - Custom component disconnection (WebSocket session end)
  - Configured Samba hostname (from settings or system config)
  - User-configurable mDNS registration toggle (new Setting)

- **Outputs:**
  - mDNS announcement of Samba server with configured hostname
  - Event via eventBus when mDNS registration succeeds/fails
  - Clean removal of mDNS announcement on component disconnect or timeout
  - Optional server->client WebSocket message confirming mDNS status

- **Dependencies:**
  - Backend: `api/ws.go` (WebSocket handler), `dto/context.go` (HAWsComponent state), `events/event_bus.go`, `service/setting_service.go`
  - Frontend: `src/pages/settings` (HomeAssistant section)
  - Custom Component: `custom_components/srat/websocket_client.py`, `custom_components/srat/__init__.py` (zeroconf registration handler)
  - HA Documentation: [Network Discovery (mDNS/Zeroconf)](https://developers.home-assistant.io/docs/network_discovery/#mdnszeroconf), [HA Zeroconf Integration](https://www.home-assistant.io/integrations/zeroconf/)

## 📝 Task List

- [ ] Task 1: Add mDNS registration Setting to backend + database migration
- [ ] Task 2: Add new WebSocket message type `mDnsRegister` to dto/events with payload schema
- [ ] Task 3: Update WebSocket handler to send `mDnsRegister` message on valid `helo` handshake and on server process CLEAN event
- [ ] Task 4: Implement disconnection timeout logic (30-sec goroutine) and send disable message
- [ ] Task 5: Emit mDNS events to eventBus for logging/diagnostics
- [ ] Task 6: Add "Enable mDNS Registration" toggle to Frontend HomeAssistant settings panel
- [ ] Task 7: Implement conditional disable + tooltip when component is not connected
- [ ] Task 8: Update custom component `websocket_client.py` to parse `mDnsRegister` messages
- [ ] Task 9: Implement zeroconf registration in custom component `__init__.py` (enable/disable)
- [ ] Task 10: Unit testing (WebSocket message handler, backend logic, settings)
- [ ] Task 11: Integration testing (custom component registration, timeout behavior, disconnection recovery)
- [ ] Task 12: Documentation (mDNS feature overview, HA zeroconf integration, troubleshooting)
- [ ] Task 13: Code review and QA

## 🧠 Implementation Notes (Copilot Context)

**Architecture:**

1. **Backend WebSocket Message Handler:**
   - New WebSocket message type: `mDnsRegister` (sent by backend to custom component)
   - Payload includes: `hostname`, `port`, `enabled` (boolean flag from settings)
   - Backend **does not** directly manage zeroconf; Home Assistant core handles that
   - Backend sends message when:
     - Valid `helo` handshake received from custom component (if mDNS enabled in settings)
     - Server process `CLEAN` event fires (state recovery)
     - Component disconnects or timeout occurs → send `enabled: false` to trigger deregistration

2. **Custom Component mDNS Handler:**
   - Update `websocket_client.py` to parse `mDnsRegister` messages from server
   - Update `__init__.py` to implement registration logic:
     - When `enabled: true` → use Home Assistant's native zeroconf integration (via async_zeroconf)
     - Register Samba service with hostname from message payload
     - When `enabled: false` or timeout → deregister from zeroconf
   - Home Assistant core manages all zeroconf state; custom component just triggers register/deregister

3. **Disconnection & Timeout:**
   - When WebSocket `/ws` session ends, backend starts 30-second timeout goroutine
   - If no new `helo` within 30s, backend sends `mDnsRegister` with `enabled: false`
   - Custom component receives disable message and deregisters from Home Assistant zeroconf
   - If new `helo` arrives before timeout, cancel the timeout and send re-enable message

4. **Server Process CLEAN Event:**
   - When eventBus fires `ServerProcess` Type `CLEAN` event:
     - Check current mDNS setting from SettingService
     - If custom component is connected, send `mDnsRegister` message with current state
     - Useful for state recovery after config reload

5. **Frontend Setting:**
   - New toggle under "HomeAssistant" settings: "Enable mDNS Registration"
   - Field depends on `HAWsComponent` connectivity state
   - **Disabled (grayed + tooltip) when:**
     - No custom component connection active
     - Custom component version is incompatible
   - **Tooltip text:** "Enable to announce this Samba server on the local network via Home Assistant. Requires an active Home Assistant add-on connection."

**Key Decisions:**

- **Backend does NOT use zeroconf library** — Home Assistant core manages mDNS/zeroconf integration
- Backend only sends **enable/disable messages** to custom component; component handles registration
- mDNS **disabled by default** to avoid network noise in multi-server setups
- **30-second timeout** chosen for graceful disconnection handling (balances responsiveness vs. flaky networks)
- **Server hostname** from settings is the mDNS service name (no additional config needed)
- mDNS port defaults to SMB port (445) in registration unless user specifies custom port
- Use **contextual logging** (slog with context) for all mDNS-related messages in backend

**Constraints:**

- mDNS registration only works when custom component is actively connected to backend
- Feature is Home Assistant-specific; desktop clients will continue discovering via standard SMB discovery
- Hostname must be valid for mDNS (alphanumeric + hyphens); frontend should validate/sanitize
- Custom component must be running on a Home Assistant instance with zeroconf support

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/service/setting_service.go` — Add mDNS toggle to Setting schema
- [ ] `TODO: backend/src/dto/webevent_type.go` — Define `mDnsRegister` message type with payload (hostname, port, enabled)
- [ ] `TODO: backend/src/api/ws.go` — Send `mDnsRegister` message on valid `helo` handshake
- [ ] `TODO: backend/src/api/ws.go` — Implement disconnection timeout goroutine (30-sec) to send disable message
- [ ] `TODO: backend/src/dto/context.go` — Track HAWsComponent disconnection timestamp for timeout logic
- [ ] `TODO: backend/src/events/event_bus.go` — Add new event types: `mdns_enabled`, `mdns_disabled` (for diagnostics)
- [ ] `TODO: backend/src/service/server_process_service.go` — On CLEAN event, re-send current mDNS status to component
- [ ] `FIXME: frontend/src/pages/settings/AppConfigurationPanel.tsx` — Add conditional mDNS toggle under HomeAssistant section based on HAWsComponent connection state
- [ ] `TODO: custom_components/srat/websocket_client.py` — Parse incoming `mDnsRegister` WebSocket messages
- [ ] `TODO: custom_components/srat/__init__.py` — Implement zeroconf registration/deregistration using Home Assistant's async_zeroconf helper
- [ ] `FIXME: custom_components/srat/manifest.json` — Verify dependencies (zeroconf should be managed by HA core, not custom component)
