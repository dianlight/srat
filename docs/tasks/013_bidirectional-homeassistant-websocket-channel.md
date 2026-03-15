# [FEATURE]: Bidirectional Home Assistant WebSocket Channel

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** _TBD_

## 🎯 Objective

Enable bidirectional communication on the existing WebSocket channel between the SRAT server and the Home Assistant custom component. Today the channel is effectively server-to-client only: the backend pushes events and the custom component listens. This task introduces the first client-to-server protocol message, `helo`, so the custom component can announce that it is connected and report its version. That allows the server to know when a specific integration build is online and creates the foundation for future commands and status exchanges from Home Assistant back to SRAT. This work depends on task `012`, but the protocol and handshake pieces can be developed in parallel as long as the final implementation remains WebSocket-only and does not reintroduce SSE-era behavior.

## 🛠️ Technical Specifications

- **Inputs:**
  - Existing `/ws` endpoint in `backend/src/api/ws.go`
  - Existing custom component WebSocket client in `custom_components/srat/websocket_client.py`
  - Custom component version from `custom_components/srat/manifest.json`
  - Existing SRAT WebSocket event model, which currently emits SSE-formatted text frames over WebSocket
- **Outputs:**
  - A bidirectional WebSocket protocol contract that supports client-to-server messages
  - Initial `helo` message sent by the custom component immediately after connect, including at minimum the integration version and component identity
  - Backend handling that records, logs, or exposes the connected custom component version/state for future automation and diagnostics
  - Automated tests covering both the backend handshake path and the custom component connect path
- **Dependencies:**
  - `docs/tasks/012_remove-sse-code-preserve-websocket.md` — related transport cleanup; final result must stay WS-only
  - `backend/src/api/ws.go` — current WebSocket upgrade and outbound event loop
  - `backend/src/server/ws/ws.go` — shared WS message sender types
  - `backend/src/service/ha_ws_service.go` — likely backend integration point for HA-specific connection state
  - `backend/src/dto/context.go` — possible location for runtime HA component connection metadata
  - `backend/src/dto/welcome.go` and `backend/src/dto/webevent_map.go` — existing outbound protocol context (`hello` already exists server-to-client)
  - `custom_components/srat/websocket_client.py` — custom component WS lifecycle and message parsing
  - `custom_components/srat/__init__.py` — WS client setup during integration startup
  - `custom_components/srat/manifest.json` — integration version source
  - `backend/src/api/ws_test.go`, `backend/src/api/ws_sender_test.go`, `custom_components/tests/test_init.py` — existing test coverage starting points

## 📝 Task List

- [ ] Task 1: Define the bidirectional WebSocket protocol contract for Home Assistant client messages, starting with a `helo` payload schema and clear separation from the existing server-to-client `hello` event
- [ ] Task 2: Extend the backend WebSocket handler to read and validate inbound client frames while preserving existing outbound event streaming and ping/pong behavior
- [ ] Task 3: Add backend handling for `helo` that stores or exposes the connected custom component identity/version and logs the connection using the existing HA integration path
- [ ] Task 4: Update the Home Assistant custom component to send `helo` immediately after a successful WebSocket connection using the integration version from `manifest.json`
- [ ] Task 5: Add backend and custom component tests for successful handshake, malformed payload handling, and reconnect behavior
- [ ] Task 6: Integration and documentation — document the new client-to-server protocol message and note the dependency/interaction with task `012`

## 🧠 Implementation Notes (Copilot Context)

### Current protocol shape

The backend currently pushes SSE-style text frames over WebSocket, for example:

- `event: hello`
- `data: {...}`

The custom component parses those frames in `SRATWebSocketClient._parse_ws_message()`. There is not yet a corresponding inbound protocol for frames sent from the custom component back to the server.

### First inbound message

Start with a dedicated `helo` client message rather than reusing the existing outbound `hello` event. The names are intentionally close but must remain unambiguous:

- `hello` = existing server-to-client welcome event
- `helo` = new custom-component-to-server handshake message

Suggested minimum payload fields:

- `type`: `"helo"`
- `component`: `"srat"`
- `version`: integration version from `custom_components/srat/manifest.json`

Optional follow-up fields if implementation needs them:

- `ha_version`
- `entry_id`
- `capabilities`

### Backend design constraints

- Preserve the existing outbound broadcaster flow in `backend/src/api/ws.go`
- Add inbound frame reading without blocking or breaking keepalive behavior
- Do not tie the first implementation to SSE concepts; task `012` is removing the remaining SSE-specific transport leftovers
- Keep protocol parsing explicit and testable; avoid baking ad-hoc JSON parsing directly into unrelated services if a small DTO or handler abstraction would make the message contract clearer

### Custom component design constraints

- Send `helo` only after the WebSocket is fully connected
- Re-send `helo` after reconnect, because the backend should treat each WS session as a fresh registration
- Keep listener registration and sensor update behavior unchanged for this task; the goal is to add the first upstream message, not redesign the coordinator

### Testing guidance

Backend coverage should include:

- WebSocket connection still upgrades successfully
- Existing server `hello` event still reaches the client
- A client-sent `helo` message is accepted and processed
- Invalid inbound payloads do not crash the handler and are logged/ignored appropriately

Custom component coverage should include:

- `async_connect()` sends `helo` on successful connection
- reconnect logic re-sends `helo`
- connection setup still succeeds when handshake send is mocked

## 🔗 Code References & TODOs

- [ ] `docs/tasks/012_remove-sse-code-preserve-websocket.md` — related dependency; keep the final transport design WS-only
- [ ] `backend/src/api/ws.go` — add inbound frame reading and handshake routing
- [ ] `backend/src/server/ws/ws.go` — extend shared WS protocol helpers if needed
- [ ] `backend/src/service/ha_ws_service.go` — evaluate as the runtime owner for HA-side WS connection metadata
- [ ] `backend/src/dto/context.go` — decide whether to store connected custom component version/state here or in a dedicated runtime structure
- [ ] `backend/src/dto/welcome.go` — existing outbound `hello` event context
- [ ] `backend/src/dto/webevent_map.go` — confirm outbound event names remain unchanged
- [ ] `custom_components/srat/websocket_client.py` — send `helo` after connect and after reconnect
- [ ] `custom_components/srat/__init__.py` — ensure startup flow still initializes WS client correctly
- [ ] `custom_components/srat/manifest.json` — source of integration version for handshake payload
- [ ] `backend/src/api/ws_test.go` — extend WS handler integration tests for inbound messages
- [ ] `backend/src/api/ws_sender_test.go` — keep existing outbound protocol expectations intact
- [ ] `custom_components/tests/test_init.py` — extend setup tests for handshake send behavior
