"""The SRAT (SambaNAS REST Administration Tool) integration."""

from __future__ import annotations

import asyncio
import logging

import aiohttp
from homeassistant.config_entries import ConfigEntry
from homeassistant.const import Platform
from homeassistant.core import HomeAssistant
from homeassistant.exceptions import ConfigEntryNotReady
from homeassistant.helpers.aiohttp_client import async_get_clientsession

from .const import CONF_HOST, CONF_PORT, DOMAIN, WS_RECONNECT_INTERVAL
from .coordinator import SRATDataCoordinator
from .websocket_client import SRATWebSocketClient

_LOGGER = logging.getLogger(__name__)

PLATFORMS: list[Platform] = [Platform.SENSOR]

type SRATConfigEntry = ConfigEntry


class SRATData:
    """Runtime data for the SRAT integration."""

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        ws_client: SRATWebSocketClient,
    ) -> None:
        """Initialize runtime data."""
        self.coordinator = coordinator
        self.ws_client = ws_client


async def async_setup_entry(hass: HomeAssistant, entry: SRATConfigEntry) -> bool:
    """Set up SRAT from a config entry."""
    host = entry.data[CONF_HOST]
    port = entry.data[CONF_PORT]

    session = async_get_clientsession(hass)

    # Verify the SRAT API is reachable
    try:
        async with asyncio.timeout(10):
            async with session.get(f"http://{host}:{port}/health") as resp:
                if resp.status != 200:
                    raise ConfigEntryNotReady(f"SRAT API returned status {resp.status}")
    except (aiohttp.ClientError, TimeoutError) as err:
        raise ConfigEntryNotReady(f"Cannot connect to SRAT at {host}:{port}") from err

    # Create WebSocket client for real-time updates
    ws_client = SRATWebSocketClient(
        hass=hass,
        host=host,
        port=port,
        reconnect_interval=WS_RECONNECT_INTERVAL,
    )

    # Create data coordinator
    coordinator = SRATDataCoordinator(
        hass=hass,
        host=host,
        port=port,
        ws_client=ws_client,
        session=session,
    )

    # Fetch initial data
    await coordinator.async_config_entry_first_refresh()

    # Start WebSocket connection
    await ws_client.async_connect()

    entry.runtime_data = SRATData(
        coordinator=coordinator,
        ws_client=ws_client,
    )

    await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)

    entry.async_on_unload(entry.add_update_listener(_async_update_listener))

    return True


async def async_unload_entry(hass: HomeAssistant, entry: SRATConfigEntry) -> bool:
    """Unload a SRAT config entry."""
    if unload_ok := await hass.config_entries.async_unload_platforms(entry, PLATFORMS):
        # Disconnect WebSocket
        await entry.runtime_data.ws_client.async_disconnect()

    return unload_ok


async def _async_update_listener(hass: HomeAssistant, entry: SRATConfigEntry) -> None:
    """Handle options update."""
    await hass.config_entries.async_reload(entry.entry_id)
