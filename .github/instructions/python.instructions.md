<!-- DOCTOC SKIP -->

---

description: "Python coding conventions for the SRAT Home Assistant custom component"
applyTo: "**/*.py"

---

# Python Development Instructions

## Python Version & Imports

- **Target**: Python 3.12+ (as required by Home Assistant 2025.x)
- **Always** start every `.py` file with `from __future__ import annotations` for PEP 604 union syntax (`X | None`) and forward references
- Use modern type syntax: `list[str]`, `dict[str, Any]`, `tuple[int, ...]` — not `List`, `Dict`, `Tuple` from `typing`
- Import `Any`, `Callable`, etc. from `typing`; import `Generator`, `Callable`, `Sequence` from `collections.abc`
- Use `type` statement for type aliases (Python 3.12+): `type SRATConfigEntry = ConfigEntry`

## General Conventions

- Follow **PEP 8** and **PEP 257** (Google-style docstrings)
- Use 4-space indentation
- All functions, methods, and classes **must** have type hints for parameters and return values
- All public functions and classes **must** have docstrings
- Use `logging.getLogger(__name__)` for module-level loggers (never `print()`)
- Prefer `contextlib.suppress(ExceptionType)` over bare `try/except: pass`
- Handle edge cases: empty inputs, `None` values, invalid data types
- Write concise, idiomatic code; break complex functions into smaller helpers

## Code Style & Formatting

- **Formatter**: `ruff format` (replaces Black)
- **Linter**: `ruff check` with rules: A, ASYNC, B, C4, D, E, F, I, ICN, LOG, N, RUF, S, SIM, T20, UP, W
- **Type checker**: `mypy` with `--strict` subset (see `custom_components/pyproject.toml`)
- Line length managed by formatter (no manual wrapping needed)
- Use `isort`-compatible import ordering (via ruff's `I` rule): stdlib → third-party → first-party
- First-party import root: `custom_components.srat`

## Home Assistant Integration Patterns

### Module Structure

Every HA custom component lives in `custom_components/<domain>/` and requires:

```
custom_components/srat/
├── __init__.py          # async_setup_entry / async_unload_entry
├── config_flow.py       # ConfigFlow subclass
├── const.py             # DOMAIN, config keys, defaults
├── coordinator.py       # DataUpdateCoordinator subclass
├── manifest.json        # Integration metadata (version injected at build)
├── sensor.py            # SensorEntity subclasses
├── strings.json         # UI strings (English, canonical)
├── translations/        # i18n files (one per language)
└── websocket_client.py  # Real-time WebSocket client
```

### Entry Points (`__init__.py`)

```python
async def async_setup_entry(hass: HomeAssistant, entry: SRATConfigEntry) -> bool:
    """Set up SRAT from a config entry."""
    ...

async def async_unload_entry(hass: HomeAssistant, entry: SRATConfigEntry) -> bool:
    """Unload a SRAT config entry."""
    ...
```

- Use `async_get_clientsession(hass)` for HTTP/WS — never create your own `aiohttp.ClientSession`
- Raise `ConfigEntryNotReady` if the backend is unreachable during setup
- Store runtime data in `entry.runtime_data` (not `hass.data`)
- Forward platform setup: `await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)`

### Config Flow (`config_flow.py`)

- Subclass `ConfigFlow` with `domain=DOMAIN` (requires `# type: ignore[call-arg]` for HA metaclass)
- Implement `async_step_user` for manual configuration (host + port)
- Implement `async_step_hassio(self, discovery_info: HassioServiceInfo)` for Supervisor autodiscovery
- Validate connectivity before creating entries (call `/health` endpoint)
- Use `self.async_set_unique_id(...)` + `self._abort_if_unique_id_configured()`
- Define form schemas with `voluptuous` (`vol.Schema`)
- Use `after_dependencies: ["hassio"]` in manifest (NOT `dependencies`) to avoid forcing hassio setup

### Data Coordinator (`coordinator.py`)

- Subclass `DataUpdateCoordinator[dict[str, Any]]`
- **WebSocket-only**: set `update_interval=None` (no REST polling)
- Register WebSocket event listeners for data channels
- Use `@callback` decorator for synchronous event handlers
- Call `self.async_set_updated_data(self.data)` to push updates to entities
- Initialize data keys to `None` — sensors report *unavailable* until first event

### Sensor Entities (`sensor.py`)

- Subclass both `CoordinatorEntity[SRATDataCoordinator]` and `SensorEntity`
- Set `_attr_has_entity_name = True` on the base class
- Use `_attr_*` class variables for static attributes (`name`, `icon`, `unique_id`, etc.)
- Implement `native_value` as a `@property` — return `None` for unavailable
- Implement `extra_state_attributes` as a `@property` returning `dict[str, Any]`
- Use `DeviceInfo` with `identifiers={(DOMAIN, entry.entry_id)}`
- For dynamic entities (per-disk, per-partition): create in `async_setup_entry` based on coordinator data

### WebSocket Client (`websocket_client.py`)

- Use `async_get_clientsession(hass)` → `session.ws_connect()` for WebSocket
- Authenticate with `X-Remote-User-Id: homeassistant` header (per `ha_middleware.go`)
- Enable `heartbeat=30` for ping/pong keep-alive
- Implement auto-reconnect loop with configurable interval
- Parse SSE-formatted text frames: `id:`, `event:`, `data:` lines
- Use `hass.async_create_background_task()` for the listen loop
- Register typed listeners: `register_listener(event_type, callback) -> unregister_fn`
- Event types (from `webevent_type.go`): `hello`, `updating`, `volumes`, `heartbeat`, `shares`, `dirty_data_tracker`, `smart_test_status`, `error`

## Error Handling

- Use `try/except` with specific exceptions; avoid bare `except:`
- Log exceptions with `_LOGGER.exception(...)` for full traceback
- Return `None` from sensor `native_value` when data is unavailable (HA shows as "unavailable")
- For WebSocket: catch `aiohttp.ClientError`, `TimeoutError`, `asyncio.CancelledError`
- Use `contextlib.suppress(asyncio.CancelledError)` when cancelling tasks

## Async Patterns

- All I/O operations **must** be `async`
- Use `asyncio.timeout(seconds)` (not `async_timeout`) for timeouts
- Use `async with` for context managers (`session.get(...)`, `session.ws_connect(...)`)
- Background tasks: `hass.async_create_background_task(coro, name)`
- Cancellation: set a flag, cancel the task, suppress `CancelledError`

## Testing

### Framework & Tools

- **Test runner**: `pytest` with `pytest-homeassistant-custom-component`
- **Async**: `pytest-asyncio` with `asyncio_mode = "auto"`
- **Coverage**: `pytest-cov` — run `make test-ci` in `custom_components/`
- Tests live in `custom_components/tests/` (not inside `srat/`)
- Run from repo root: `cd custom_components && make test`

### Test Structure

```python
"""Tests for SRAT config flow."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

from homeassistant.core import HomeAssistant
from homeassistant.data_entry_flow import FlowResultType
import pytest
```

### Key Patterns

- Use `auto_enable_custom_integrations` fixture (autouse in `conftest.py`)
- Mock `aiohttp.ClientSession.get()` with `MagicMock(spec=aiohttp.ClientSession)` — **not** `AsyncMock` for the `.get()` method itself (use `MagicMock(return_value=async_context_manager)`)
- Use `patch("custom_components.srat.module.function")` for targeted mocking
- Test both success and error paths (connection errors, invalid data, empty responses)
- For config flow: assert `FlowResultType.FORM`, `FlowResultType.CREATE_ENTRY`, `FlowResultType.ABORT`
- For sensors: verify `native_value`, `extra_state_attributes`, and unavailable (`None`) states

### Running Tests

```bash
cd custom_components
make test          # Run all tests
make test-ci       # Run with coverage (generates coverage.xml)
make check         # Full check: format + lint + typecheck + test
```

## Documentation

- Module-level docstrings explain the file's purpose
- Class docstrings describe responsibility and data flow
- Method docstrings use Google style with `Args:` / `Returns:` / `Raises:` sections when non-trivial
- Use reStructuredText in docstrings for code references: `` ``event_type`` ``
- In-line comments explain *why*, not *what*
- Reference backend source files when relevant: `# See backend/src/api/ws.go`

## Security

- Never hardcode secrets or credentials
- Use ruff's `S` (bandit) rules for security scanning
- Validate all external data before use (`isinstance` checks)
- Sanitize user input used in entity IDs (`_sanitize_id()` pattern)
- WebSocket auth uses header-based identity, not tokens

## Manifest & HACS

- `manifest.json` version: `0.0.0` in source, injected as `YYYY.MM.PATCH` at build time
- HACS distribution: `hacs.json` at repo root with `zip_release: true`, `filename: srat.zip`
- Use `after_dependencies: ["hassio"]` — never `dependencies: ["hassio"]`
- `iot_class: local_push` (WebSocket push, no polling)
- `integration_type: hub` (single integration managing multiple entities)

## Makefile Targets

All custom component tooling runs via `custom_components/Makefile`:

| Target       | Description                                      |
| ------------ | ------------------------------------------------ |
| `check`      | Run format-check + lint + typecheck + test        |
| `lint`       | Run `ruff check`                                  |
| `format`     | Run `ruff format` (auto-fix)                      |
| `typecheck`  | Run `mypy`                                        |
| `test`       | Run `pytest`                                      |
| `test-ci`    | Run `pytest` with coverage (generates XML)        |
| `fix`        | Run `ruff check --fix` + `ruff format`            |
| `install`    | Install dev deps (auto-detects Alpine apk)        |
| `install-pip`| Install dev deps via pip only                     |
| `clean`      | Remove caches and build artifacts                 |
