"""Data coordinator for the SRAT integration.

All sensor data is received exclusively via the WebSocket connection.
No REST API polling is used.  The ``heartbeat`` event carries
``HealthPing`` which embeds ``samba_status``, ``samba_process_status``,
and ``disk_health``.  The ``volumes`` event carries the disk list.
"""

from __future__ import annotations

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

        # Register WebSocket listeners for real-time updates
        # Event types match backend/src/dto/webevent_type.go string values
        ws_client.register_listener("volumes", self._on_volumes)
        ws_client.register_listener("heartbeat", self._on_heartbeat)

    async def _async_update_data(self) -> dict[str, Any]:
        """Return current data (no REST polling)."""
        return self.data

    @callback
    def _on_volumes(self, data: Any) -> None:
        """Handle ``volumes`` event (list of disks)."""
        self.data["disks"] = data if isinstance(data, list) else None
        self.async_set_updated_data(self.data)

    @callback
    def _on_heartbeat(self, data: Any) -> None:
        """Handle ``heartbeat`` event (``HealthPing``).

        ``HealthPing`` carries embedded fields::

            samba_status         → SambaStatus
            samba_process_status → ServerProcessStatus
            disk_health          → DiskHealth
        """
        if not isinstance(data, dict):
            return
        self.data["samba_status"] = data.get("samba_status")
        self.data["process_status"] = data.get("samba_process_status")
        self.data["disk_health"] = data.get("disk_health")
        self.async_set_updated_data(self.data)
