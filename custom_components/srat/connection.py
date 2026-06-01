"""Connection helpers for reaching the SRAT backend from Home Assistant."""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from homeassistant.components.hassio import get_supervisor_client

from .const import SUPERVISOR_GATEWAY_HOST

if TYPE_CHECKING:
    from homeassistant.core import HomeAssistant


def homeassistant_auth_headers() -> dict[str, str]:
    """Return the headers required by SRAT's Home Assistant middleware."""
    return {"X-Remote-User-Id": "homeassistant"}


def iter_connection_hosts(host: str, addon_slug: str | None = None) -> tuple[str, ...]:
    """Return candidate hosts in preferred connection order.

    Home Assistant Supervisor discovery commonly exposes add-on hostnames such as
    ``core-...`` or ``local-...``. In the test environment those names are not a
    reliable websocket target from Home Assistant Core, while the Supervisor
    gateway host is. Prefer the gateway first for Supervisor-managed add-ons, but
    retain the discovered hostname as a fallback.
    """
    normalized_host = host.strip()
    if not normalized_host:
        return (SUPERVISOR_GATEWAY_HOST,)

    if addon_slug or normalized_host.startswith(("core-", "local-")):
        return tuple(dict.fromkeys((SUPERVISOR_GATEWAY_HOST, normalized_host)))

    return (normalized_host,)


async def resolve_supervisor_addon_endpoint(
    hass: HomeAssistant,
    addon_slug: str,
    *,
    fallback_port: int,
) -> tuple[str, int]:
    """Resolve addon host/port from Supervisor API.

    This is intended for integrations running in the same Supervisor environment.
    For remote/manual configurations, explicit host/port should be used instead.
    """
    addon_info: Any = await get_supervisor_client(hass).addons.addon_info(addon_slug)

    hostname = str(getattr(addon_info, "hostname", "") or "").strip()
    ip_host = str(getattr(addon_info, "ip_address", "") or "").strip()

    # Prefer addon hostname for Supervisor-internal routing; retain gateway/IP
    # fallback paths in iter_connection_hosts().
    host = hostname or ip_host or SUPERVISOR_GATEWAY_HOST

    ingress_port = getattr(addon_info, "ingress_port", None)
    if isinstance(ingress_port, int) and ingress_port > 0:
        return host, ingress_port

    network = getattr(addon_info, "network", None)
    if isinstance(network, dict):
        net_port = network.get("3000/tcp")
        if isinstance(net_port, int) and net_port > 0:
            return host, net_port

    return host, fallback_port
