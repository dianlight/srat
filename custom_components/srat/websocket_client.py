"""WebSocket client for SRAT real-time updates."""

from __future__ import annotations

import asyncio
import json
import logging
from collections.abc import Callable
from typing import Any

import aiohttp
from homeassistant.core import HomeAssistant, callback
from homeassistant.helpers.aiohttp_client import async_get_clientsession

_LOGGER = logging.getLogger(__name__)

# SSE event types from the SRAT backend
EVENT_DISK_STATUS = "disk_status"
EVENT_SAMBA_STATUS = "samba_status"
EVENT_SAMBA_PROCESS = "samba_process"
EVENT_DISK_HEALTH = "disk_health"
EVENT_VOLUME_STATUS = "volume_status"


class SRATWebSocketClient:
    """WebSocket/SSE client for SRAT real-time updates.

    SRAT uses Server-Sent Events (SSE) for real-time push notifications.
    This client connects to the SSE endpoint and dispatches events.
    """

    def __init__(
        self,
        hass: HomeAssistant,
        host: str,
        port: int,
        reconnect_interval: int = 5,
    ) -> None:
        """Initialize the SRAT WebSocket/SSE client."""
        self._hass = hass
        self._host = host
        self._port = port
        self._reconnect_interval = reconnect_interval
        self._listeners: dict[str, list[Callable[[dict[str, Any]], None]]] = {}
        self._task: asyncio.Task | None = None
        self._connected = False
        self._should_reconnect = True

    @property
    def connected(self) -> bool:
        """Return whether the client is connected."""
        return self._connected

    @callback
    def register_listener(
        self, event_type: str, listener: Callable[[dict[str, Any]], None]
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
        """Start the SSE connection."""
        self._should_reconnect = True
        self._task = self._hass.async_create_background_task(
            self._listen_loop(), "srat_sse_listener"
        )

    async def async_disconnect(self) -> None:
        """Disconnect from SSE."""
        self._should_reconnect = False
        if self._task and not self._task.done():
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        self._connected = False

    async def _listen_loop(self) -> None:
        """Main SSE listen loop with automatic reconnection."""
        session = async_get_clientsession(self._hass)
        url = f"http://{self._host}:{self._port}/events"

        while self._should_reconnect:
            try:
                _LOGGER.debug("Connecting to SRAT SSE at %s", url)
                async with session.get(
                    url,
                    headers={"Accept": "text/event-stream"},
                    timeout=aiohttp.ClientTimeout(total=None, sock_read=None),
                ) as resp:
                    if resp.status != 200:
                        _LOGGER.warning(
                            "SRAT SSE returned status %s, retrying in %ss",
                            resp.status,
                            self._reconnect_interval,
                        )
                        await asyncio.sleep(self._reconnect_interval)
                        continue

                    self._connected = True
                    _LOGGER.info("Connected to SRAT SSE at %s", url)

                    event_type = ""
                    data_lines: list[str] = []

                    async for line_bytes in resp.content:
                        line = line_bytes.decode("utf-8").rstrip("\n\r")

                        if line.startswith("event:"):
                            event_type = line[6:].strip()
                        elif line.startswith("data:"):
                            data_lines.append(line[5:].strip())
                        elif line == "" and event_type:
                            # End of event
                            data_str = "\n".join(data_lines)
                            self._dispatch_event(event_type, data_str)
                            event_type = ""
                            data_lines = []

            except asyncio.CancelledError:
                _LOGGER.debug("SRAT SSE listener cancelled")
                break
            except (aiohttp.ClientError, TimeoutError) as err:
                self._connected = False
                if self._should_reconnect:
                    _LOGGER.warning(
                        "SRAT SSE connection lost (%s), reconnecting in %ss",
                        err,
                        self._reconnect_interval,
                    )
                    await asyncio.sleep(self._reconnect_interval)

        self._connected = False

    def _dispatch_event(self, event_type: str, data: str) -> None:
        """Dispatch an SSE event to registered listeners."""
        try:
            parsed = json.loads(data) if data else {}
        except json.JSONDecodeError:
            _LOGGER.warning("Failed to parse SSE data for event %s", event_type)
            return

        listeners = self._listeners.get(event_type, [])
        for listener in listeners:
            try:
                listener(parsed)
            except Exception:
                _LOGGER.exception(
                    "Error in SSE listener for event %s", event_type
                )
