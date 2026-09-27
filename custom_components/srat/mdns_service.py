"""mDNS registration service for SRAT Samba advertisements."""

from __future__ import annotations

import asyncio
import logging
import socket
from typing import Any, cast

from homeassistant.components.zeroconf import async_get_async_instance
from homeassistant.core import HomeAssistant, callback
from zeroconf import ServiceInfo

from .websocket_client import SRATWebSocketClient

_LOGGER = logging.getLogger(__name__)


class SRATMdnsService:
    """Owns mDNS register/unregister state for Samba advertisements."""

    def __init__(self, hass: HomeAssistant, resolved_host: str | None) -> None:
        """Initialize the mDNS service."""
        self._hass = hass
        self._resolved_host = resolved_host
        self._registered_info: ServiceInfo | None = None
        self._lock = asyncio.Lock()
        self._remove_legacy: Any = None
        self._remove_current: Any = None
        self._closed = False
        self._tasks: set[asyncio.Future[None]] = set()

    async def _register_mdns(self, info: ServiceInfo) -> None:
        """Register a Zeroconf ServiceInfo with Home Assistant's shared zeroconf."""
        zc = cast(Any, await async_get_async_instance(self._hass))
        await zc.async_register_service(info, allow_name_change=True)
        _LOGGER.debug("mDNS: registered %s on port %d", info.name, info.port)

    async def _unregister_mdns(self, info: ServiceInfo) -> None:
        """Unregister a previously registered Zeroconf ServiceInfo."""
        zc = cast(Any, await async_get_async_instance(self._hass))
        await zc.async_unregister_service(info)
        _LOGGER.debug("mDNS: unregistered %s", info.name)

    @callback
    def _on_mdns_register(self, event_data: dict) -> None:
        """Handle mdns_register WebSocket events from the backend."""
        if self._closed:
            return
        enabled: bool = bool(event_data.get("enabled", False))
        hostname: str = str(event_data.get("hostname", ""))
        port: int = int(event_data.get("port", 445))

        service_type = "_smb._tcp.local."
        service_name = f"{hostname}.{service_type}"

        async def _apply() -> None:
            async with self._lock:
                if self._closed:
                    return
                if self._registered_info is not None:
                    try:
                        await self._unregister_mdns(self._registered_info)
                    except Exception:
                        _LOGGER.debug("mDNS: unregister failed (may already be gone)")
                    self._registered_info = None

                if not enabled or not hostname:
                    return

                raw_ip = (
                    getattr(self._hass.config.api, "local_ip", None)
                    or self._resolved_host
                )
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
                # Track the info before awaiting so a cancellation between
                # register and assignment cannot leak the advertisement.
                self._registered_info = info
                try:
                    await self._register_mdns(info)
                except Exception:
                    self._registered_info = None
                    _LOGGER.exception("mDNS: failed to register %s", service_name)

        task = self._hass.async_create_task(_apply())
        self._tasks.add(task)
        task.add_done_callback(self._tasks.discard)

    def register(self, ws_client: SRATWebSocketClient) -> None:
        """Subscribe to backend mDNS events."""
        self._remove_legacy = ws_client.register_listener(
            "m_dns_register", self._on_mdns_register
        )
        self._remove_current = ws_client.register_listener(
            "mdns_register", self._on_mdns_register
        )

    def unregister(self) -> None:
        """Remove backend mDNS subscriptions."""
        if self._remove_legacy is not None:
            self._remove_legacy()
            self._remove_legacy = None
        if self._remove_current is not None:
            self._remove_current()
            self._remove_current = None

    async def async_cleanup(self) -> None:
        """Deregister mDNS if it was registered."""
        self._closed = True
        pending = tuple(self._tasks)
        if pending:
            for task in pending:
                task.cancel()
            await asyncio.gather(*pending, return_exceptions=True)
            self._tasks.difference_update(pending)
        async with self._lock:
            if self._registered_info is not None:
                try:
                    await self._unregister_mdns(self._registered_info)
                except Exception:
                    _LOGGER.debug("mDNS: unregister on unload failed")
                self._registered_info = None
