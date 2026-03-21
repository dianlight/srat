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

from .connection import homeassistant_auth_headers, iter_connection_hosts
from .const import (
    CONF_ADDON_SLUG,
    CONF_HOST,
    CONF_PORT,
    DOMAIN,
    WS_RECONNECT_INTERVAL,
)
from .coordinator import SRATDataCoordinator
from .repairs import SRATRepairProxy
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
        repair_proxy: SRATRepairProxy,
    ) -> None:
        """Initialize runtime data."""
        self.coordinator = coordinator
        self.ws_client = ws_client
        self.repair_proxy = repair_proxy


async def async_setup_entry(hass: HomeAssistant, entry: SRATConfigEntry) -> bool:
    """Set up SRAT from a config entry."""
    configured_host = entry.data[CONF_HOST]
    port = entry.data[CONF_PORT]
    addon_slug = entry.data.get(CONF_ADDON_SLUG)

    session = async_get_clientsession(hass)

    resolved_host: str | None = None
    last_error: Exception | None = None

    for candidate_host in iter_connection_hosts(configured_host, addon_slug):
        try:
            async with asyncio.timeout(10):
                async with session.get(
                    f"http://{candidate_host}:{port}/api/health",
                    headers=homeassistant_auth_headers(),
                ) as resp:
                    if resp.status == 200:
                        resolved_host = candidate_host
                        break
                    last_error = ConfigEntryNotReady(
                        f"SRAT API returned status {resp.status}"
                    )
        except (aiohttp.ClientError, TimeoutError) as err:
            last_error = err

    if resolved_host is None:
        raise ConfigEntryNotReady(
            f"Cannot connect to SRAT at {configured_host}:{port}"
        ) from last_error

    # Create WebSocket client for real-time updates (sole data channel)
    ws_client = SRATWebSocketClient(
        hass=hass,
        host=configured_host,
        port=port,
        reconnect_interval=WS_RECONNECT_INTERVAL,
        addon_slug=addon_slug,
    )

    # Create data coordinator (no REST polling, WebSocket only)
    coordinator = SRATDataCoordinator(
        hass=hass,
        host=resolved_host,
        port=port,
        ws_client=ws_client,
    )

    # Start WebSocket connection — data arrives via events
    await ws_client.async_connect()

    repair_proxy = SRATRepairProxy(hass=hass, ws_client=ws_client)
    repair_proxy.register()

    # Listen for remote configuration changes and trigger integration reload
    def _on_app_config_changed(event_data: dict) -> None:
        """Handle app_config_changed event from the backend."""
        _LOGGER.info("Addon configuration changed, reloading integration")
        hass.async_create_task(hass.config_entries.async_reload(entry.entry_id))

    unregister_listener = ws_client.register_listener(
        "app_config_changed", _on_app_config_changed
    )

    entry.runtime_data = SRATData(
        coordinator=coordinator,
        ws_client=ws_client,
        repair_proxy=repair_proxy,
    )

    # Store unregister function for cleanup on unload
    def _on_unload() -> None:
        unregister_listener()

    entry.async_on_unload(_on_unload)

    await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)

    entry.async_on_unload(entry.add_update_listener(_async_update_listener))

    return True


async def async_unload_entry(hass: HomeAssistant, entry: SRATConfigEntry) -> bool:
    """Unload a SRAT config entry."""
    if unload_ok := await hass.config_entries.async_unload_platforms(entry, PLATFORMS):
        # Disconnect WebSocket
        entry.runtime_data.repair_proxy.unregister()
        await entry.runtime_data.ws_client.async_disconnect()

    return unload_ok


async def _async_update_listener(hass: HomeAssistant, entry: SRATConfigEntry) -> None:
    """Handle options update."""
    await hass.config_entries.async_reload(entry.entry_id)
