"""Data coordinator for the SRAT integration.

All sensor data is received exclusively via the WebSocket connection.
No REST API polling is used.  The ``heartbeat`` event carries
``HealthPing`` which embeds ``samba_status``, ``samba_process_status``,
and ``disk_health``.  The ``volumes`` event carries the disk list.
"""

from __future__ import annotations

from collections.abc import Callable
import logging
from typing import Any

from homeassistant.core import HomeAssistant, callback
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator

from .const import DOMAIN
from .websocket_client import SRATWebSocketClient

_LOGGER = logging.getLogger(__name__)


class SRATDataCoordinator(DataUpdateCoordinator[dict[str, Any]]):
    """Coordinator that receives all data from the SRAT WebSocket.

    No REST polling is performed.  Data arrives via two WebSocket events:

    * ``volumes`` → ``[]*Disk{}`` — disk & partition information
    * ``heartbeat`` → ``HealthPing`` — samba status, process status,
      disk health, network health, addon stats, etc.

    Until the first event of each type arrives the corresponding data
    key is ``None`` and sensors report as *unavailable*.
    """

    def __init__(
        self,
        hass: HomeAssistant,
        host: str,
        port: int,
        ws_client: SRATWebSocketClient,
    ) -> None:
        """Initialize the coordinator."""
        super().__init__(
            hass,
            _LOGGER,
            name=DOMAIN,
            # No periodic polling — data comes from WebSocket only
            update_interval=None,
        )
        self._host = host
        self._port = port
        self._ws_client = ws_client

        # Seed with empty/unavailable data
        self.data: dict[str, Any] = {
            "disks": None,
            "samba_status": None,
            "process_status": None,
            "disk_health": None,
        }

        # One-shot callbacks keyed by data slot, fired when the slot first
        # holds usable data (issue #1263: the backend only emits ``volumes``
        # on mount/unmount transitions, so platforms must defer entity
        # creation until the first payload arrives).
        self._data_ready_callbacks: dict[str, list[Callable[[], bool]]] = {}

        # Register WebSocket listeners for real-time updates
        # Event types match backend/src/dto/webevent_type.go string values
        ws_client.register_listener("volumes", self._on_volumes)
        ws_client.register_listener("heartbeat", self._on_heartbeat)

    async def _async_update_data(self) -> dict[str, Any]:
        """Return current data (no REST polling)."""
        return self.data

    @callback
    def async_when_data_ready(
        self, key: str, on_ready: Callable[[], bool]
    ) -> Callable[[], None]:
        """Register a callback fired when ``data[key]`` first holds usable data.

        The backend only emits ``volumes`` on mount/unmount transitions, so
        slots seeded ``None`` may stay empty for the whole session. Platforms
        that build dynamic entities register here instead of reading
        ``self.data`` once at setup.

        The callback must return ``True`` once it has consumed the data (the
        registration is then dropped), or ``False`` to stay registered for a
        later, more complete payload (e.g. an empty disk list).

        If the key already holds data the callback is invoked immediately.

        Returns:
            A function that unregisters the callback.
        """
        if self.data.get(key) is not None and on_ready():
            return lambda: None
        callbacks = self._data_ready_callbacks.setdefault(key, [])
        callbacks.append(on_ready)

        def _unregister() -> None:
            if on_ready in callbacks:
                callbacks.remove(on_ready)

        return _unregister

    @callback
    def _fire_data_ready(self, *keys: str) -> None:
        """Notify consumers of ``keys`` whose data just became available."""
        for key in keys:
            if self.data.get(key) is None:
                continue
            pending = self._data_ready_callbacks.pop(key, [])
            still_waiting: list[Callable[[], bool]] = []
            for cb in pending:
                if not cb():
                    still_waiting.append(cb)
            if still_waiting:
                self._data_ready_callbacks[key] = still_waiting

    @callback
    def _on_volumes(self, data: Any) -> None:
        """Handle ``volumes`` event (list of disks)."""
        self.data["disks"] = data if isinstance(data, list) else None
        self.async_set_updated_data(dict(self.data))
        self._fire_data_ready("disks")

    @callback
    def _on_heartbeat(self, data: Any) -> None:
        """Handle ``heartbeat`` event (``HealthPing``).

        ``HealthPing`` carries embedded fields::

            samba_status         → SambaStatus
            samba_process_status → ServerProcessStatus
            disk_health          → DiskHealth

        Merge (not overwrite): only present keys update state so a partial
        heartbeat never clears unrelated keys with ``None``.
        """
        if not isinstance(data, dict):
            return
        if "samba_status" in data:
            self.data["samba_status"] = data.get("samba_status")
        if "samba_process_status" in data:
            self.data["process_status"] = data.get("samba_process_status")
        if "disk_health" in data:
            self.data["disk_health"] = data.get("disk_health")
        self.async_set_updated_data(dict(self.data))
        self._fire_data_ready("samba_status", "process_status", "disk_health")
