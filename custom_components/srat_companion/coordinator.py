"""Data coordinator for SRAT Companion."""
from __future__ import annotations

import asyncio
import logging
from typing import Any

import aiohttp
from aiohttp_sse_client import client as sse_client

from homeassistant.config_entries import ConfigEntry
from homeassistant.const import CONF_HOST, CONF_PORT
from homeassistant.core import HomeAssistant
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed
from homeassistant.helpers import issue_registry as ir

from .const import DOMAIN, SSE_ENDPOINT

_LOGGER = logging.getLogger(__name__)


class SratCoordinator(DataUpdateCoordinator):
    """SRAT Companion coordinator."""

    def __init__(self, hass: HomeAssistant, entry: ConfigEntry) -> None:
        """Initialize the coordinator."""
        super().__init__(
            hass,
            _LOGGER,
            name=DOMAIN,
            update_interval=None,  # We use SSE for real-time updates
        )
        self.entry = entry
        self.host = entry.data[CONF_HOST]
        self.port = entry.data[CONF_PORT]
        self.base_url = f"http://{self.host}:{self.port}"
        self.session = async_get_clientsession(hass)
        self.sse_task: asyncio.Task | None = None
        self.connected = False

    async def _async_update_data(self) -> dict[str, Any]:
        """Update data via API polling."""
        try:
            async with self.session.get(
                f"{self.base_url}/api/status",
                timeout=aiohttp.ClientTimeout(total=10),
            ) as response:
                if response.status == 200:
                    data = await response.json()
                    self.connected = True
                    return data
                else:
                    raise UpdateFailed(f"HTTP {response.status}")
        except Exception as err:
            self.connected = False
            raise UpdateFailed(f"Error communicating with API: {err}") from err

    async def async_config_entry_first_refresh(self) -> None:
        """Perform first refresh and start SSE connection."""
        await super().async_config_entry_first_refresh()
        
        # Start SSE connection
        self.sse_task = asyncio.create_task(self._async_sse_listener())

    async def async_shutdown(self) -> None:
        """Shutdown the coordinator."""
        if self.sse_task and not self.sse_task.done():
            self.sse_task.cancel()
            try:
                await self.sse_task
            except asyncio.CancelledError:
                pass

    async def _async_sse_listener(self) -> None:
        """Listen for SSE events."""
        while True:
            try:
                _LOGGER.debug("Connecting to SSE endpoint at %s%s", self.base_url, SSE_ENDPOINT)
                
                async with sse_client.EventSource(
                    f"{self.base_url}{SSE_ENDPOINT}",
                    session=self.session,
                ) as event_source:
                    self.connected = True
                    # Clear any existing repair issues
                    ir.async_delete_issue(self.hass, DOMAIN, "connection_error")
                    
                    async for event in event_source:
                        try:
                            if event.data:
                                # Process the event
                                await self._process_sse_event(event)
                        except Exception as err:
                            _LOGGER.error("Error processing SSE event: %s", err)
                            
            except Exception as err:
                _LOGGER.error("SSE connection error: %s", err)
                self.connected = False
                
                # Create repair issue
                ir.async_create_issue(
                    self.hass,
                    DOMAIN,
                    "connection_error",
                    is_fixable=True,
                    severity=ir.IssueSeverity.ERROR,
                    translation_key="connection_error",
                    translation_placeholders={
                        "host": self.host,
                        "port": str(self.port),
                        "error": str(err),
                    },
                )
                
                # Wait before reconnecting
                await asyncio.sleep(30)

    async def _process_sse_event(self, event) -> None:
        """Process an SSE event."""
        try:
            import json
            event_data = json.loads(event.data)
            
            # Update our data with the event
            if not self.data:
                self.data = {}
            
            # Store the event in our data
            if "events" not in self.data:
                self.data["events"] = []
            
            self.data["events"].append(event_data)
            
            # Keep only the last 100 events
            if len(self.data["events"]) > 100:
                self.data["events"] = self.data["events"][-100:]
            
            # Notify listeners
            self.async_update_listeners()
            
        except Exception as err:
            _LOGGER.error("Error processing SSE event data: %s", err)

