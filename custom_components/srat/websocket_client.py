"""WebSocket client for SRAT real-time updates.

Connects to the SRAT backend WebSocket endpoint (/ws) using the gorilla/websocket
protocol.  The backend sends text frames whose payload uses SSE-style formatting
for backwards compatibility::

    id: <sequence>
    event: <event_type>
    data: <json_payload>

Authentication uses the ``X-Remote-User-Id`` header required by the HA
middleware (see ``backend/src/server/ha_middleware.go``).
"""

from __future__ import annotations

import asyncio
from collections.abc import Callable
import contextlib
import json
import logging
from pathlib import Path
from typing import Any

import aiohttp
from homeassistant.core import HomeAssistant, callback
from homeassistant.helpers.aiohttp_client import async_get_clientsession

from .connection import homeassistant_auth_headers, iter_connection_hosts

_LOGGER = logging.getLogger(__name__)

_MANIFEST_PATH = Path(__file__).with_name("manifest.json")


def _load_integration_version() -> str:
    """Load the integration version from the manifest."""
    try:
        manifest = json.loads(_MANIFEST_PATH.read_text(encoding="utf-8"))
    except (FileNotFoundError, OSError, json.JSONDecodeError):
        _LOGGER.warning("Unable to load SRAT integration version from manifest")
        return "0.0.0"

    version = manifest.get("version")
    if isinstance(version, str) and version:
        return version

    _LOGGER.warning("SRAT integration manifest version missing or invalid")
    return "0.0.0"


INTEGRATION_VERSION = _load_integration_version()


class SRATWebSocketClient:
    """WebSocket client for SRAT real-time updates.

    Connects to the ``/ws`` endpoint using the WebSocket protocol with
    automatic reconnection and ping/pong keep-alive.
    """

    def __init__(
        self,
        hass: HomeAssistant,
        host: str,
        port: int,
        reconnect_interval: int = 5,
        integration_version: str | None = None,
        addon_slug: str | None = None,
    ) -> None:
        """Initialize the SRAT WebSocket client."""
        self._hass = hass
        self._host = host
        self._port = port
        self._connection_hosts = iter_connection_hosts(host, addon_slug)
        self._reconnect_interval = reconnect_interval
        self._integration_version = integration_version or INTEGRATION_VERSION
        self._listeners: dict[str, list[Callable[[Any], None]]] = {}
        self._task: asyncio.Task | None = None
        self._connected = False
        self._should_reconnect = True
        self._ws: aiohttp.ClientWebSocketResponse | None = None
        self._send_lock = asyncio.Lock()

    @property
    def connected(self) -> bool:
        """Return whether the client is connected."""
        return self._connected

    @callback
    def register_listener(
        self, event_type: str, listener: Callable[[Any], None]
    ) -> Callable[[], None]:
        """Register a listener for a specific event type.

        Returns a callable to unregister the listener.
        """
        self._listeners.setdefault(event_type, []).append(listener)

        def _remove_listener() -> None:
            self._listeners[event_type].remove(listener)
            if not self._listeners[event_type]:
                del self._listeners[event_type]

        return _remove_listener

    async def async_connect(self) -> None:
        """Start the WebSocket connection."""
        self._should_reconnect = True
        self._task = self._hass.async_create_background_task(
            self._listen_loop(), "srat_ws_listener"
        )

    async def async_disconnect(self) -> None:
        """Disconnect from the WebSocket."""
        self._should_reconnect = False
        if self._task and not self._task.done():
            self._task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await self._task
        self._connected = False

    async def _listen_loop(self) -> None:
        """Main WebSocket listen loop with automatic reconnection."""
        session = async_get_clientsession(self._hass)
        headers = homeassistant_auth_headers()

        while self._should_reconnect:
            reconnect_reason = "WebSocket closed"

            for candidate_host in self._connection_hosts:
                url = f"ws://{candidate_host}:{self._port}/ws"
                try:
                    _LOGGER.debug("Connecting to SRAT WebSocket at %s", url)
                    async with session.ws_connect(
                        url,
                        headers=headers,
                        heartbeat=30,
                        autoclose=True,
                        autoping=True,
                    ) as ws:
                        self._connected = True
                        self._ws = ws
                        _LOGGER.info("Connected to SRAT WebSocket at %s", url)
                        await self._send_helo(ws)

                        async for msg in ws:
                            if msg.type == aiohttp.WSMsgType.TEXT:
                                self._parse_ws_message(msg.data)
                            elif msg.type == aiohttp.WSMsgType.ERROR:
                                reconnect_reason = str(ws.exception())
                                _LOGGER.warning(
                                    "SRAT WebSocket error: %s", ws.exception()
                                )
                                break
                            elif msg.type in (
                                aiohttp.WSMsgType.CLOSE,
                                aiohttp.WSMsgType.CLOSING,
                                aiohttp.WSMsgType.CLOSED,
                            ):
                                _LOGGER.debug("SRAT WebSocket closed")
                                break

                    break
                except asyncio.CancelledError:
                    _LOGGER.debug("SRAT WebSocket listener cancelled")
                    self._should_reconnect = False
                    break
                except (
                    aiohttp.ClientError,
                    ConnectionError,
                    OSError,
                    RuntimeError,
                    TimeoutError,
                ) as err:
                    reconnect_reason = str(err)
                    self._connected = False
                    self._ws = None
                    _LOGGER.debug(
                        "SRAT WebSocket connection failed for %s: %s",
                        url,
                        err,
                    )
                    continue
                finally:
                    self._ws = None

            self._connected = False
            if not self._should_reconnect:
                break

            _LOGGER.warning(
                "SRAT WebSocket connection lost (%s), reconnecting in %ss",
                reconnect_reason,
                self._reconnect_interval,
            )
            await asyncio.sleep(self._reconnect_interval)

        self._connected = False

    async def async_send_repair_lifecycle_event(
        self,
        *,
        repair_id: str,
        status: str,
        command_id: str | None = None,
        error: str | None = None,
        details: dict[str, Any] | None = None,
    ) -> None:
        """Send a repair lifecycle event to the backend over the active WebSocket."""
        if not self._connected or self._ws is None:
            _LOGGER.debug(
                "Skipping repair lifecycle send while disconnected: %s/%s",
                repair_id,
                status,
            )
            return

        payload: dict[str, Any] = {
            "type": "repair_lifecycle",
            "repair_id": repair_id,
            "status": status,
        }
        if command_id:
            payload["command_id"] = command_id
        if error:
            payload["error"] = error
        if details:
            payload["details"] = details

        async with self._send_lock:
            try:
                await self._ws.send_json(payload)
            except (RuntimeError, ConnectionError, aiohttp.ClientError):
                _LOGGER.exception(
                    "Failed to send repair lifecycle event for %s", repair_id
                )

    async def _send_helo(self, ws: aiohttp.ClientWebSocketResponse) -> None:
        """Send the initial client-to-server handshake message."""
        payload = {
            "type": "helo",
            "component": "srat",
            "version": self._integration_version,
        }
        _LOGGER.info(
            "Sending SRAT WebSocket helo with integration version %s",
            self._integration_version,
        )
        await ws.send_json(payload)

    def _parse_ws_message(self, raw: str) -> None:
        """Parse an SSE-formatted WebSocket text frame.

        The backend sends text frames formatted as::

            id: 42
            event: volumes
            data: { ... }

        """
        event_type = ""
        data_lines: list[str] = []

        for line in raw.split("\n"):
            line = line.rstrip("\r")
            if line.startswith("event:"):
                event_type = line[6:].strip()
            elif line.startswith("data:"):
                data_lines.append(line[5:].strip())

        if event_type and data_lines:
            self._dispatch_event(event_type, "\n".join(data_lines))

    def _dispatch_event(self, event_type: str, data: str) -> None:
        """Dispatch a parsed event to registered listeners."""
        try:
            parsed: Any = json.loads(data) if data else {}
        except json.JSONDecodeError:
            _LOGGER.warning("Failed to parse data for event %s", event_type)
            return

        listeners = self._listeners.get(event_type, [])
        for listener in listeners:
            try:
                listener(parsed)
            except Exception:
                _LOGGER.exception("Error in listener for event %s", event_type)
