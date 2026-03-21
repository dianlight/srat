"""Connection helpers for reaching the SRAT backend from Home Assistant."""

from __future__ import annotations

from .const import SUPERVISOR_GATEWAY_HOST


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
