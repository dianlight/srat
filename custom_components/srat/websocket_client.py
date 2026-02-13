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
from typing import Any

import aiohttp
from homeassistant.core import HomeAssistant, callback
from homeassistant.helpers.aiohttp_client import async_get_clientsession

_LOGGER = logging.getLogger(__name__)


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
    ) -> None:
        """Initialize the SRAT WebSocket client."""
        self._hass = hass
        self._host = host
        self._port = port
        self._reconnect_interval = reconnect_interval
        self._listeners: dict[str, list[Callable[[Any], None]]] = {}
        self._task: asyncio.Task | None = None
        self._connected = False
        self._should_reconnect = True

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
        url = f"ws://{self._host}:{self._port}/ws"
        # Auth header required by backend/src/server/ha_middleware.go
        headers = {"X-Remote-User-Id": "homeassistant"}

        while self._should_reconnect:
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
                    _LOGGER.info("Connected to SRAT WebSocket at %s", url)

                    async for msg in ws:
                        if msg.type == aiohttp.WSMsgType.TEXT:
                            self._parse_ws_message(msg.data)
                        elif msg.type == aiohttp.WSMsgType.ERROR:
                            _LOGGER.warning("SRAT WebSocket error: %s", ws.exception())
                            break
                        elif msg.type in (
                            aiohttp.WSMsgType.CLOSE,
                            aiohttp.WSMsgType.CLOSING,
                            aiohttp.WSMsgType.CLOSED,
                        ):
                            _LOGGER.debug("SRAT WebSocket closed")
                            break

            except asyncio.CancelledError:
                _LOGGER.debug("SRAT WebSocket listener cancelled")
                break
            except (aiohttp.ClientError, TimeoutError) as err:
                self._connected = False
                if self._should_reconnect:
                    _LOGGER.warning(
                        "SRAT WebSocket connection lost (%s), reconnecting in %ss",
                        err,
                        self._reconnect_interval,
                    )
                    await asyncio.sleep(self._reconnect_interval)

        self._connected = False

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
