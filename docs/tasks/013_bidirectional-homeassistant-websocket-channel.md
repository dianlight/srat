# [FEATURE]: Bidirectional Home Assistant WebSocket Channel

**Target Repo:** `srat`  **Status:** ✅ Complete  **Issue Link:** https://github.com/dianlight/srat/issues/508

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

- [x] Task 1: Define the bidirectional WebSocket protocol contract for Home Assistant client messages, starting with a `helo` payload schema and clear separation from the existing server-to-client `hello` event
- [x] Task 2: Extend the backend WebSocket handler to read and validate inbound client frames while preserving existing outbound event streaming and ping/pong behavior
- [x] Task 3: Add backend handling for `helo` that stores or exposes the connected custom component identity/version and logs the connection using the existing HA integration path
- [x] Task 4: Update the Home Assistant custom component to send `helo` immediately after a successful WebSocket connection using the integration version from `manifest.json`
- [x] Task 5: Add backend and custom component tests for successful handshake, malformed payload handling, and reconnect behavior
- [x] Task 6: Integration and documentation — document the new client-to-server protocol message and note the dependency/interaction with task `012`

## 🧠 Implementation Notes (Copilot Context)

### Current protocol shape
**Branch Strategy:** Created `feature/bidirectional-websocket-ha` for isolated development
### Agreed Implementation Plan

**Phase 1: Protocol Definition & Backend Inbound Handler**
- Create new DTO `HeloMessage` in `backend/src/dto/helo.go` with fields: `type`, `component`, `version`, optional `ha_version`, `entry_id`
- Add `helo` to `WebEventType` enum in `backend/src/dto/webevent_map.go` to maintain event registry
- Modify `ws.go` handler loop to read inbound text frames in addition to pings, routing messages by `type` field
- Extract message routing logic into a handler function that dispatches to protocol-specific handlers (future-proof for multiple message types)
- Add panic recovery and validation to prevent malformed payloads from crashing the listener

**Phase 2: Backend Connection State Tracking**
- Extend `HaWsService` or create lightweight state holder in `dto/context.go` to track connected component version/identity
- Store version info when `helo` is received; log successful handshake with component and version
- Design for future multi-client support (optional unique client ID in handshake)

**Phase 3: Custom Component Implementation**
- Modify `websocket_client.py` to send `helo` immediately after WebSocket connect succeeds
- Extract version from `manifest.json` and include in payload
- Log handshake send action for visibility
- Ensure reconnect logic re-sends `helo` as part of recovery

**Phase 4: Testing**
- Backend: Extend `ws_test.go` to verify inbound frame handling, `helo` parsing, and state storage (without breaking existing tests)
- Custom component: Extend `test_init.py` to verify handshake send on connect, reconnect re-send behavior
- Both: Cover malformed/invalid frames; verify ping/pong and existing events continue normally

**Phase 5: Documentation & Polish**
- Update task docs with discovered protocol details
- Add code comments explaining bidirectional frame separation
- Verify no regression in existing WebSocket event flow

### Progress Notes

- Completed Task 1 by introducing `dto.ClientMessageEnvelope` and `dto.HeloMessage` in `backend/src/dto/helo.go`.
- The inbound contract explicitly reserves `type: "helo"` for Home Assistant handshake frames and rejects `"hello"` so the new client-to-server message stays separate from the existing outbound welcome event.
- Added focused DTO coverage in `backend/src/dto/helo_test.go` for JSON round-trip, validation, and envelope parsing.
- Validation passed: `cd /workspaces/srat/backend/src && go test ./dto -run 'TestHeloMessage|TestClientMessageEnvelope'`.
- Completed Task 2 by adding a dedicated inbound read loop in `backend/src/api/ws.go` that parses text frames, validates `helo`, ignores malformed/unsupported payloads, and exits cleanly on disconnect while preserving the existing outbound broadcaster goroutine and ping/pong keepalive.
- Added handler coverage in `backend/src/api/ws_test.go` proving that a valid inbound `helo` and malformed JSON payload both leave outbound `hello`/`updating` delivery intact.
- Validation passed: `cd /workspaces/srat/backend/src && go test ./api -run 'TestWsHandlerSuite'`.
- Completed Task 3 by storing accepted Home Assistant handshake metadata in `dto.ContextState.HAWsComponent`, logging successful `helo` registration with the reported component/version, and clearing the metadata when the WebSocket session disconnects.
- Added runtime state coverage in `backend/src/api/ws_test.go` verifying that valid `helo` frames populate `ContextState`, malformed payloads leave it unset, and disconnect clears the active component registration.
- Validation passed: `cd /workspaces/srat/backend/src && go test ./api ./dto -run 'TestWsHandlerSuite|TestWsMessageSenderSuite|TestContextState'`.
- Completed Task 4 by loading the integration version from `custom_components/srat/manifest.json` in `custom_components/srat/websocket_client.py` and sending a `{"type":"helo","component":"srat","version":...}` payload immediately after each successful WebSocket connection.
- The client now logs the outbound handshake send action and reuses the same connect path on reconnect, so `helo` will be re-sent automatically for each fresh session without changing listener/coordinator behavior.
- Added focused custom-component coverage in `custom_components/tests/test_websocket_client.py` and re-ran `custom_components/tests/test_init.py` to confirm setup/unload still works.
- Validation passed: `cd /workspaces/srat/custom_components && pytest tests/test_init.py tests/test_websocket_client.py -q` (4 passed).
- Completed Task 5 by extending `custom_components/tests/test_websocket_client.py` with reconnect coverage that verifies `helo` is re-sent on a second successful WebSocket session, while the existing backend `backend/src/api/ws_test.go` coverage already validates successful inbound `helo` handling and malformed payload resilience.
- Focused validation passed: `cd /workspaces/srat/backend/src && go test ./api -run 'TestWsHandlerSuite' && cd /workspaces/srat/custom_components && pytest tests/test_init.py tests/test_websocket_client.py -q` (Go ok, Python 5 passed).
- Completed Task 6 by documenting the WS-only Home Assistant transport and the `helo` client-to-server handshake in `docs/HOME_ASSISTANT_INTEGRATION.md` and `docs/EVENT_DRIVEN_ARCHITECTURE.md`, explicitly tying the final protocol shape to task `012`'s SSE removal.
- Documentation validation passed: `make docs-validate`.

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
- Client connection metadata (e.g. version) can be stored in the existing HA integration context or in a new runtime structure, but should be easily accessible for future use in automation rules or diagnostics
- Multiple clients are not expected in the near term, but the protocol design should not preclude supporting multiple connections in the future if needed (e.g. by including a unique client ID in the handshake)
- Client messages must be validated and sanitized to prevent malformed payloads from crashing the handler; invalid messages should be logged and ignored without disrupting the connection
- Client messages should not trigger any existing server-to-client events or state changes until the handshake is successfully completed and the client identity is known
- The handshake should be re-sent on reconnect, as the backend should treat each WebSocket session as a fresh registration even if the same client connects multiple times
- The backend should log successful handshakes with the reported component version for visibility into what clients are connecting and for easier debugging in the future when more message types are added
- The backend should be designed to easily accommodate additional client-to-server message types in the future, using a clear routing or handler pattern based on the `type` field in the payload


### Custom component design constraints

- Send `helo` only after the WebSocket is fully connected
- Re-send `helo` after reconnect, because the backend should treat each WS session as a fresh registration
- Keep listener registration and sensor update behavior unchanged for this task; the goal is to add the first upstream message, not redesign the coordinator
- The custom component should log the handshake send action for visibility, including the version being reported
- The custom component should handle any connection errors gracefully and attempt to reconnect as it does today, ensuring that the handshake will be re-attempted on the next successful connection

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
- [x] `backend/src/api/ws.go` — inbound frame reading and validation now runs alongside outbound streaming
- [ ] `backend/src/server/ws/ws.go` — extend shared WS protocol helpers if needed
- [ ] `backend/src/service/ha_ws_service.go` — evaluate as the runtime owner for HA-side WS connection metadata
- [x] `backend/src/dto/context.go` — runtime owner for active HA WebSocket client metadata (`HAWsComponent`)
- [x] `backend/src/dto/helo.go` — inbound Home Assistant client message contract (`ClientMessageEnvelope`, `HeloMessage`)
- [ ] `backend/src/dto/welcome.go` — existing outbound `hello` event context
- [ ] `backend/src/dto/webevent_map.go` — confirm outbound event names remain unchanged
- [x] `backend/src/dto/helo_test.go` — protocol contract tests for JSON shape and validation
- [ ] `custom_components/srat/websocket_client.py` — send `helo` after connect and after reconnect
- [x] `custom_components/srat/websocket_client.py` — sends `helo` after each successful connect using the manifest version
- [x] `custom_components/srat/__init__.py` — startup flow still initializes the WS client without extra wiring changes
- [x] `custom_components/srat/manifest.json` — used as the source of the reported integration version
- [ ] `backend/src/api/ws_test.go` — extend WS handler integration tests for inbound messages
- [x] `backend/src/api/ws_test.go` — inbound frame tests cover valid `helo` and malformed payload resilience
- [x] `backend/src/api/ws_sender_test.go` — constructor updated to inject `ContextState` while preserving outbound protocol expectations
- [x] `custom_components/tests/test_websocket_client.py` — focused client test for outbound `helo` on successful connect
- [x] `custom_components/tests/test_websocket_client.py` — now also covers `helo` re-send after reconnect
- [x] `custom_components/tests/test_init.py` — existing setup/unload tests revalidated against the new WS client behavior
- [x] `docs/HOME_ASSISTANT_INTEGRATION.md` — documents the `helo` handshake and WS-only transport
- [x] `docs/EVENT_DRIVEN_ARCHITECTURE.md` — notes the inbound `helo` handshake and task `012` transport cleanup dependency
