"""Tests for mDNS / Zeroconf registration in the SRAT integration."""

from __future__ import annotations

from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import aiohttp
from homeassistant.core import HomeAssistant
from homeassistant.setup import async_setup_component
from pytest_homeassistant_custom_component.common import MockConfigEntry

from custom_components.srat.const import DOMAIN


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
            "custom_components.srat.async_get_zeroconf",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        # Simulate the backend sending a m_dns_register event with enabled=True
        mdns_handler = ws._listeners.get("m_dns_register")
        assert mdns_handler is not None, "m_dns_register listener was not registered"

        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": True})
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
            "custom_components.srat.async_get_zeroconf",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        mdns_handler = ws._listeners.get("m_dns_register")
        assert mdns_handler is not None

        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": False})
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
            "custom_components.srat.async_get_zeroconf",
            return_value=mock_zeroconf,
        ),
    ):
        ws = _make_ws_mock()
        mock_ws_cls.return_value = ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        mdns_handler = ws._listeners.get("m_dns_register")
        assert mdns_handler is not None

        # First registration
        mdns_handler({"hostname": "sambanas", "port": 445, "enabled": True})
        await hass.async_block_till_done()

        # Second registration — previous should be unregistered first
        mdns_handler({"hostname": "newhost", "port": 445, "enabled": True})
        await hass.async_block_till_done()

    assert mock_zeroconf.async_unregister_service.call_count >= 1
    assert mock_zeroconf.async_register_service.call_count == 2
