"""Tests for mDNS / Zeroconf registration in the SRAT integration."""

from __future__ import annotations

import asyncio
from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import aiohttp
from homeassistant.core import HomeAssistant
from homeassistant.setup import async_setup_component
from pytest_homeassistant_custom_component.common import MockConfigEntry
from zeroconf import ServiceInfo

from custom_components.srat.const import DOMAIN
from custom_components.srat.mdns_service import SRATMdnsService


def _mock_session(status: int = 200) -> MagicMock:
    """Create a mock aiohttp session with a given response status."""
    mock_resp = AsyncMock()
    mock_resp.status = status

    mock_ctx = AsyncMock()
    mock_ctx.__aenter__ = AsyncMock(return_value=mock_resp)
    mock_ctx.__aexit__ = AsyncMock(return_value=False)

    session = MagicMock(spec=aiohttp.ClientSession)
    session.get = MagicMock(return_value=mock_ctx)
    return session


def _make_ws_mock() -> AsyncMock:
    """Return a minimal WebSocket client mock with listener capture support."""
    ws = AsyncMock()
    ws.async_connect = AsyncMock()
    ws.async_disconnect = AsyncMock()
    _listeners: dict[str, Any] = {}

    def _register(event: str, cb: Any) -> Any:
        _listeners[event] = cb
        return lambda: _listeners.pop(event, None)

    ws.register_listener = MagicMock(side_effect=_register)
    ws._listeners = _listeners
    return ws


async def test_mdns_registers_on_enabled_event(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test that a m_dns_register event with enabled=True registers a Zeroconf service."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_mdns_register",
    )
    entry.add_to_hass(hass)

    mock_zeroconf = AsyncMock()
    mock_zeroconf.async_register_service = AsyncMock()
    mock_zeroconf.async_unregister_service = AsyncMock()

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch("custom_components.srat.SRATWebSocketClient") as mock_ws_cls,
        patch(
            "custom_components.srat.mdns_service.async_get_async_instance",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        # Simulate the backend sending a mdns_register event with enabled=True
        mdns_handler = ws._listeners.get("mdns_register")
        assert mdns_handler is not None, "mdns_register listener was not registered"

        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": True})
        await hass.async_block_till_done()

        assert await hass.config_entries.async_unload(entry.entry_id)
        await hass.async_block_till_done()

    mock_zeroconf.async_register_service.assert_called_once()
    service_info = mock_zeroconf.async_register_service.call_args[0][0]
    assert service_info.name == "sambanas._smb._tcp.local."
    assert service_info.port == 445


async def test_mdns_skips_registration_when_disabled(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test that a m_dns_register event with enabled=False does not register a service."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_mdns_disabled",
    )
    entry.add_to_hass(hass)

    mock_zeroconf = AsyncMock()
    mock_zeroconf.async_register_service = AsyncMock()

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch("custom_components.srat.SRATWebSocketClient") as mock_ws_cls,
        patch(
            "custom_components.srat.mdns_service.async_get_async_instance",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        mdns_handler = ws._listeners.get("mdns_register")
        assert mdns_handler is not None

        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": False})
        await hass.async_block_till_done()

        assert await hass.config_entries.async_unload(entry.entry_id)
        await hass.async_block_till_done()

    mock_zeroconf.async_register_service.assert_not_called()


async def test_mdns_unregisters_previous_on_new_event(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test that a second m_dns_register event unregisters the previous service first."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_mdns_rereg",
    )
    entry.add_to_hass(hass)

    mock_zeroconf = AsyncMock()
    mock_zeroconf.async_register_service = AsyncMock()
    mock_zeroconf.async_unregister_service = AsyncMock()

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch("custom_components.srat.SRATWebSocketClient") as mock_ws_cls,
        patch(
            "custom_components.srat.mdns_service.async_get_async_instance",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        mdns_handler = ws._listeners.get("mdns_register")
        assert mdns_handler is not None

        # First registration
        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": True})
        await hass.async_block_till_done()

        # Second registration — previous should be unregistered first
        mdns_handler({"hostname": "newhost", "port": 445, "enabled": True})
        await hass.async_block_till_done()

        assert await hass.config_entries.async_unload(entry.entry_id)
        await hass.async_block_till_done()

    assert mock_zeroconf.async_unregister_service.call_count >= 1
    assert mock_zeroconf.async_register_service.call_count == 2


async def test_mdns_registers_legacy_event_name_for_backward_compatibility(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test that legacy m_dns_register events are still handled."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_mdns_legacy_event",
    )
    entry.add_to_hass(hass)

    mock_zeroconf = AsyncMock()
    mock_zeroconf.async_register_service = AsyncMock()

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch("custom_components.srat.SRATWebSocketClient") as mock_ws_cls,
        patch(
            "custom_components.srat.mdns_service.async_get_async_instance",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        legacy_handler = ws._listeners.get("m_dns_register")
        assert legacy_handler is not None, "m_dns_register listener was not registered"

        legacy_handler({"hostname": "legacy", "port": 445, "enabled": True})
        await hass.async_block_till_done()

        assert await hass.config_entries.async_unload(entry.entry_id)
        await hass.async_block_till_done()

    mock_zeroconf.async_register_service.assert_called_once()
    service_info = mock_zeroconf.async_register_service.call_args[0][0]
    assert service_info.name == "legacy._smb._tcp.local."


async def test_mdns_event_after_unload_does_not_register(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test a late mdns_register event delivered after unload is ignored."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_mdns_late_event",
    )
    entry.add_to_hass(hass)

    mock_zeroconf = AsyncMock()
    mock_zeroconf.async_register_service = AsyncMock()
    mock_zeroconf.async_unregister_service = AsyncMock()

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch("custom_components.srat.SRATWebSocketClient") as mock_ws_cls,
        patch(
            "custom_components.srat.mdns_service.async_get_async_instance",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        mdns_handler = ws._listeners.get("mdns_register")
        assert mdns_handler is not None

        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": True})
        await hass.async_block_till_done()
        assert mock_zeroconf.async_register_service.call_count == 1

        assert await hass.config_entries.async_unload(entry.entry_id)
        await hass.async_block_till_done()

        # A race can still deliver a captured event after cleanup closed the
        # service; it must not re-register the advertisement.
        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": True})
        await hass.async_block_till_done()

    assert mock_zeroconf.async_register_service.call_count == 1
    assert mock_zeroconf.async_unregister_service.call_count >= 1


async def test_mdns_cleanup_cancels_inflight_apply(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test unload cancels an in-flight _apply task instead of leaking it."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_mdns_inflight",
    )
    entry.add_to_hass(hass)

    started = asyncio.Event()

    async def _blocking_register(*args: Any, **kwargs: Any) -> None:
        """Simulate a register call that hangs until cancelled."""
        started.set()
        await asyncio.Event().wait()  # never set: only cancellation ends this

    mock_zeroconf = AsyncMock()
    mock_zeroconf.async_register_service = AsyncMock(side_effect=_blocking_register)
    mock_zeroconf.async_unregister_service = AsyncMock()

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch("custom_components.srat.SRATWebSocketClient") as mock_ws_cls,
        patch(
            "custom_components.srat.mdns_service.async_get_async_instance",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        mdns_handler = ws._listeners.get("mdns_register")
        assert mdns_handler is not None

        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": True})
        await asyncio.wait_for(started.wait(), timeout=5)

        # Unload must cancel the hanging task and finish (no deadlock).
        assert await asyncio.wait_for(
            hass.config_entries.async_unload(entry.entry_id), timeout=10
        )
        await hass.async_block_till_done()

    # The tracked registration is cleaned up by async_cleanup under the lock.
    assert mock_zeroconf.async_unregister_service.call_count >= 1


def _dummy_service_info() -> ServiceInfo:
    """Build an offline ServiceInfo for state-seeding tests."""
    return ServiceInfo(
        type_="_smb._tcp.local.",
        name="seed._smb._tcp.local.",
        addresses=[b"\xc0\xa8\x01\x64"],
        port=445,
    )


def _is_closed(service: SRATMdnsService) -> bool:
    """Read the closed flag through a call boundary (defeats mypy narrowing)."""
    return service._closed


def _registered_info(service: SRATMdnsService) -> ServiceInfo | None:
    """Read the tracked info through a call boundary (defeats mypy narrowing)."""
    return service._registered_info


async def test_apply_returns_when_closed_inside_lock(hass: HomeAssistant) -> None:
    """An _apply task that loses the race to cleanup must not register."""
    service = SRATMdnsService(hass=hass, resolved_host="192.168.1.100")

    await service._lock.acquire()
    try:
        service._on_mdns_register(
            {"hostname": "sambanas", "port": 445, "enabled": True}
        )
        service._closed = True
    finally:
        service._lock.release()

    with patch.object(
        SRATMdnsService, "_register_mdns", new=AsyncMock()
    ) as mock_register:
        await hass.async_block_till_done()

    mock_register.assert_not_called()
    assert service._registered_info is None


async def test_unregister_failure_still_registers_new(
    hass: HomeAssistant, caplog: Any
) -> None:
    """A stale unregister failure must not block the new registration."""
    service = SRATMdnsService(hass=hass, resolved_host="192.168.1.100")
    service._registered_info = _dummy_service_info()

    with (
        patch.object(
            SRATMdnsService,
            "_unregister_mdns",
            new=AsyncMock(side_effect=RuntimeError("gone")),
        ) as mock_unregister,
        patch.object(
            SRATMdnsService, "_register_mdns", new=AsyncMock()
        ) as mock_register,
        patch("socket.inet_aton", return_value=b"\xc0\xa8\x01\x64"),
    ):
        service._on_mdns_register(
            {"hostname": "sambanas", "port": 445, "enabled": True}
        )
        await hass.async_block_till_done()

    mock_unregister.assert_awaited_once()
    mock_register.assert_awaited_once()
    assert service._registered_info is not None
    assert service._registered_info.name == "sambanas._smb._tcp.local."


async def test_unparseable_ip_skips_registration(
    hass: HomeAssistant, caplog: Any
) -> None:
    """An unparseable IP must warn and skip registration."""
    service = SRATMdnsService(hass=hass, resolved_host="not-an-ip")

    with (
        patch.object(
            SRATMdnsService, "_register_mdns", new=AsyncMock()
        ) as mock_register,
        patch("socket.inet_aton", side_effect=OSError("bad ip")),
        caplog.at_level("WARNING", logger="custom_components.srat.mdns_service"),
    ):
        service._on_mdns_register(
            {"hostname": "sambanas", "port": 445, "enabled": True}
        )
        await hass.async_block_till_done()

    mock_register.assert_not_called()
    assert service._registered_info is None
    assert "cannot convert IP" in caplog.text


async def test_register_failure_resets_state(hass: HomeAssistant, caplog: Any) -> None:
    """A failed registration must clear the tracked info and log the error."""
    service = SRATMdnsService(hass=hass, resolved_host="192.168.1.100")

    with (
        patch.object(
            SRATMdnsService,
            "_register_mdns",
            new=AsyncMock(side_effect=RuntimeError("denied")),
        ),
        patch("socket.inet_aton", return_value=b"\xc0\xa8\x01\x64"),
        caplog.at_level("ERROR", logger="custom_components.srat.mdns_service"),
    ):
        service._on_mdns_register(
            {"hostname": "sambanas", "port": 445, "enabled": True}
        )
        await hass.async_block_till_done()

    assert service._registered_info is None
    assert "failed to register" in caplog.text


async def test_cleanup_unregister_failure_clears_state(
    hass: HomeAssistant, caplog: Any
) -> None:
    """A failing unload-time unregister must still clear the tracked info."""
    service = SRATMdnsService(hass=hass, resolved_host="192.168.1.100")
    service._registered_info = _dummy_service_info()

    with (
        patch.object(
            SRATMdnsService,
            "_unregister_mdns",
            new=AsyncMock(side_effect=RuntimeError("gone")),
        ) as mock_unregister,
        caplog.at_level("DEBUG", logger="custom_components.srat.mdns_service"),
    ):
        await service.async_cleanup()

    mock_unregister.assert_awaited_once()
    assert _registered_info(service) is None
    assert _is_closed(service)
    assert "unregister on unload failed" in caplog.text
