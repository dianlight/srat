"""The SRAT (SambaNAS REST Administration Tool) integration."""

from __future__ import annotations

import asyncio
import contextlib
import logging
import socket
from typing import Any, cast

import aiohttp
from homeassistant.components.zeroconf import async_get_async_instance
from homeassistant.config_entries import ConfigEntry
from homeassistant.const import Platform
from homeassistant.core import HomeAssistant
from homeassistant.exceptions import ConfigEntryNotReady
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from zeroconf import ServiceInfo

from .connection import (
    homeassistant_auth_headers,
    iter_connection_hosts,
    resolve_supervisor_addon_endpoint,
)
from .const import (
    ADDON_API_PORT,
    CONF_ADDON_SLUG,
    CONF_HOST,
    CONF_HOST_AUTO,
    CONF_PORT,
    CONF_PORT_AUTO,
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
    configured_port = entry.data[CONF_PORT]
    addon_slug = entry.data.get(CONF_ADDON_SLUG)

    auto_supervisor_endpoint = bool(
        addon_slug
        and configured_host == CONF_HOST_AUTO
        and configured_port == CONF_PORT_AUTO
    )

    session = async_get_clientsession(hass)

    resolved_host: str | None = None
    resolved_port: int | None = None
    last_error: Exception | None = None

    if addon_slug:
        try:
            resolved_host, resolved_port = await resolve_supervisor_addon_endpoint(
                hass,
                cast(str, addon_slug),
                fallback_port=ADDON_API_PORT,
            )
        except Exception as err:
            last_error = err

    candidate_ports = [configured_port]
    if (
        addon_slug
        and configured_port != ADDON_API_PORT
        and not auto_supervisor_endpoint
    ):
        # Backward compatibility for older config entries that stored
        # an ingress/dynamic port; try the stable addon API port first.
        candidate_ports = [ADDON_API_PORT, configured_port]

    candidate_host_seed = (
        resolved_host if resolved_host is not None else configured_host
    )
    for candidate_host in iter_connection_hosts(candidate_host_seed, addon_slug):
        for candidate_port in candidate_ports:
            if resolved_port is not None:
                candidate_port = resolved_port
            try:
                async with asyncio.timeout(10):
                    async with session.get(
                        f"http://{candidate_host}:{candidate_port}/api/health",
                        headers=homeassistant_auth_headers(),
                    ) as resp:
                        if resp.status == 200:
                            resolved_host = candidate_host
                            resolved_port = candidate_port
                            break
                        last_error = ConfigEntryNotReady(
                            f"SRAT API returned status {resp.status}"
                        )
            except (aiohttp.ClientError, TimeoutError) as err:
                last_error = err

        if resolved_host is not None:
            break

    if resolved_host is None or resolved_port is None:
        raise ConfigEntryNotReady(
            f"Cannot connect to SRAT at {configured_host}:{configured_port}"
        ) from last_error

    if addon_slug and not auto_supervisor_endpoint:
        _LOGGER.info(
            "Migrating SRAT config entry to supervisor auto endpoint mode",
        )
        hass.config_entries.async_update_entry(
            entry,
            data={
                **dict(entry.data),
                CONF_HOST: CONF_HOST_AUTO,
                CONF_PORT: CONF_PORT_AUTO,
                CONF_ADDON_SLUG: addon_slug,
            },
        )

    async def _resolve_ws_endpoint() -> tuple[str, int]:
        """Resolve current WebSocket endpoint for Supervisor-managed entries."""
        return await resolve_supervisor_addon_endpoint(
            hass,
            cast(str, addon_slug),
            fallback_port=ADDON_API_PORT,
        )

    # Create WebSocket client for real-time updates (sole data channel)
    # Use the validated host that passed /api/health checks to keep WebSocket
    # connectivity aligned with the working backend target.
    ws_client = SRATWebSocketClient(
        hass=hass,
        host=resolved_host,
        port=resolved_port,
        reconnect_interval=WS_RECONNECT_INTERVAL,
        addon_slug=None,
        endpoint_resolver=_resolve_ws_endpoint if addon_slug else None,
    )

    # Create data coordinator (no REST polling, WebSocket only)
    coordinator = SRATDataCoordinator(
        hass=hass,
        host=resolved_host,
        port=resolved_port,
        ws_client=ws_client,
    )

    # Start WebSocket connection — data arrives via events
    await ws_client.async_connect()

    async def _ws_watchdog_loop() -> None:
        """Ensure WS listener task keeps running and self-heals if needed."""
        while True:
            await asyncio.sleep(15)
            await ws_client.async_ensure_running()

    ws_watchdog_task = hass.async_create_background_task(
        _ws_watchdog_loop(),
        "srat_ws_watchdog",
    )

    repair_proxy = SRATRepairProxy(hass=hass, ws_client=ws_client)
    repair_proxy.register()

    # Listen for remote configuration changes and trigger integration reload
    def _on_app_config_changed(event_data: dict) -> None:
        """Handle app_config_changed event from the backend."""
        _LOGGER.info("Addon configuration changed, reloading integration")
        hass.async_create_task(hass.config_entries.async_reload(entry.entry_id))

    unregister_app_config_listener = ws_client.register_listener(
        "app_config_changed", _on_app_config_changed
    )

    # mDNS / Zeroconf registration state — tracks the currently registered ServiceInfo
    _mdns_registered_info: ServiceInfo | None = None

    async def _register_mdns(info: ServiceInfo) -> None:
        """Register a Zeroconf ServiceInfo with Home Assistant's shared zeroconf."""
        zc = cast(Any, await async_get_async_instance(hass))
        await zc.async_register_service(info, allow_name_change=True)
        _LOGGER.debug("mDNS: registered %s on port %d", info.name, info.port)

    async def _unregister_mdns(info: ServiceInfo) -> None:
        """Unregister a previously registered Zeroconf ServiceInfo."""
        zc = cast(Any, await async_get_async_instance(hass))
        await zc.async_unregister_service(info)
        _LOGGER.debug("mDNS: unregistered %s", info.name)

    def _on_mdns_register(event_data: dict) -> None:
        """Handle m_dns_register WebSocket events from the backend.

        The backend sends this event on every new component connection so the
        custom component can register or unregister the Samba server via mDNS.
        """
        nonlocal _mdns_registered_info

        enabled: bool = bool(event_data.get("enabled", False))
        hostname: str = str(event_data.get("hostname", ""))
        port: int = int(event_data.get("port", 445))

        service_type = "_smb._tcp.local."
        service_name = f"{hostname}.{service_type}"

        async def _apply() -> None:
            nonlocal _mdns_registered_info

            # Unregister any previously registered service
            if _mdns_registered_info is not None:
                try:
                    await _unregister_mdns(_mdns_registered_info)
                except Exception:
                    _LOGGER.debug("mDNS: unregister failed (may already be gone)")
                _mdns_registered_info = None

            if not enabled or not hostname:
                return

            # Resolve an IPv4 address for the service advertisement.
            # Prefer the HA API local IP; fall back to the resolved SRAT host.
            raw_ip = getattr(hass.config.api, "local_ip", None) or resolved_host
            try:
                packed_ip = socket.inet_aton(str(raw_ip))
            except OSError:
                _LOGGER.warning("mDNS: cannot convert IP %r to packed form", raw_ip)
                return

            info = ServiceInfo(
                type_=service_type,
                name=service_name,
                addresses=[packed_ip],
                port=port,
                properties={"path": "/"},
            )
            try:
                await _register_mdns(info)
                _mdns_registered_info = info
            except Exception:
                _LOGGER.exception("mDNS: failed to register %s", service_name)

        hass.async_create_task(_apply())

    unregister_mdns_listener_legacy = ws_client.register_listener(
        "m_dns_register", _on_mdns_register
    )
    unregister_mdns_listener = ws_client.register_listener(
        "mdns_register", _on_mdns_register
    )

    entry.runtime_data = SRATData(
        coordinator=coordinator,
        ws_client=ws_client,
        repair_proxy=repair_proxy,
    )

    # Store unregister functions for cleanup on unload
    async def _on_unload() -> None:
        unregister_app_config_listener()
        unregister_mdns_listener_legacy()
        unregister_mdns_listener()
        ws_watchdog_task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await ws_watchdog_task
        # Deregister mDNS if it was registered
        if _mdns_registered_info is not None:
            try:
                await _unregister_mdns(_mdns_registered_info)
            except Exception:
                _LOGGER.debug("mDNS: unregister on unload failed")

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
