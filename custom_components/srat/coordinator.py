"""Data coordinator for the SRAT integration."""

from __future__ import annotations

import asyncio
from datetime import timedelta
import logging
from typing import Any

import aiohttp
from homeassistant.core import HomeAssistant
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed

from .const import DOMAIN, SENSOR_UPDATE_INTERVAL
from .websocket_client import SRATWebSocketClient

_LOGGER = logging.getLogger(__name__)


class SRATDataCoordinator(DataUpdateCoordinator[dict[str, Any]]):
    """Coordinator to fetch data from SRAT API and receive real-time updates."""

    def __init__(
        self,
        hass: HomeAssistant,
        host: str,
        port: int,
        ws_client: SRATWebSocketClient,
        session: aiohttp.ClientSession,
    ) -> None:
        """Initialize the coordinator."""
        super().__init__(
            hass,
            _LOGGER,
            name=DOMAIN,
            update_interval=timedelta(seconds=SENSOR_UPDATE_INTERVAL),
        )
        self._host = host
        self._port = port
        self._ws_client = ws_client
        self._session = session
        self._base_url = f"http://{host}:{port}"

        # Register WebSocket listeners for real-time updates
        # Event types match backend/src/dto/webevent_type.go string values
        ws_client.register_listener("volumes", self._on_disk_update)
        ws_client.register_listener("heartbeat", self._on_heartbeat_update)

    async def _async_update_data(self) -> dict[str, Any]:
        """Fetch data from the SRAT REST API."""
        data: dict[str, Any] = {
            "disks": [],
            "samba_status": None,
            "process_status": None,
            "disk_health": None,
        }

        try:
            async with asyncio.timeout(10):
                # Fetch disk data
                async with self._session.get(f"{self._base_url}/disks") as resp:
                    if resp.status == 200:
                        data["disks"] = await resp.json()

                # Fetch samba status
                async with self._session.get(f"{self._base_url}/samba/status") as resp:
                    if resp.status == 200:
                        data["samba_status"] = await resp.json()

                # Fetch process status
                async with self._session.get(f"{self._base_url}/samba/process") as resp:
                    if resp.status == 200:
                        data["process_status"] = await resp.json()

                # Fetch disk health
                async with self._session.get(f"{self._base_url}/health/disks") as resp:
                    if resp.status == 200:
                        data["disk_health"] = await resp.json()

        except (aiohttp.ClientError, TimeoutError) as err:
            raise UpdateFailed(f"Error communicating with SRAT API: {err}") from err

        return data

    def _on_disk_update(self, data: Any) -> None:
        """Handle volumes event from WebSocket (list of disks)."""
        if self.data is not None:
            # The "volumes" event sends []*Disk{} directly
            if isinstance(data, list):
                self.data["disks"] = data
            else:
                self.data["disks"] = data.get("disks", self.data.get("disks", []))
            self.async_set_updated_data(self.data)

    def _on_heartbeat_update(self, data: Any) -> None:
        """Handle heartbeat event from WebSocket (HealthPing)."""
        if self.data is not None:
            self.data["disk_health"] = data
            self.async_set_updated_data(self.data)
