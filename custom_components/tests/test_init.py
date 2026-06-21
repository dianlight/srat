"""Tests for SRAT integration setup and teardown."""

from __future__ import annotations

from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import aiohttp
from homeassistant.config_entries import ConfigEntryState
from homeassistant.core import HomeAssistant
from homeassistant.setup import async_setup_component
from pytest_homeassistant_custom_component.common import MockConfigEntry

from custom_components.srat.connection import homeassistant_auth_headers
from custom_components.srat.const import (
    ADDON_API_PORT,
    CONF_HOST_AUTO,
    CONF_PORT_AUTO,
    DOMAIN,
    SUPERVISOR_GATEWAY_HOST,
)


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


async def test_setup_entry(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test successful setup of a config entry."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_entry_id",
    )
    entry.add_to_hass(hass)

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch(
            "custom_components.srat.SRATWebSocketClient",
        ) as mock_ws_cls,
    ):
        mock_ws = AsyncMock()
        mock_ws.register_listener = lambda event, cb: lambda: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.LOADED


async def test_setup_entry_sends_homeassistant_auth_header(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test setup health checks include the HA auth header."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_entry_headers",
    )
    entry.add_to_hass(hass)

    mock_session = _mock_session(200)
    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=mock_session,
        ),
        patch(
            "custom_components.srat.SRATWebSocketClient",
        ) as mock_ws_cls,
    ):
        mock_ws = AsyncMock()
        mock_ws.register_listener = lambda event, cb: lambda: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.LOADED
    mock_session.get.assert_called_once_with(
        f"http://{mock_config_entry_data['host']}:{mock_config_entry_data['port']}/api/health",
        headers=homeassistant_auth_headers(),
    )


async def test_setup_entry_health_check_fails(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test that setup retries when health check returns non-200."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_entry_fail",
    )
    entry.add_to_hass(hass)

    with patch(
        "custom_components.srat.async_get_clientsession",
        return_value=_mock_session(503),
    ):
        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.SETUP_RETRY


async def test_setup_entry_prefers_supervisor_gateway_host(
    hass: HomeAssistant,
) -> None:
    """Test Supervisor-discovered entries prefer the gateway host for SRAT."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data={
            "host": "local-sambanas2",
            "port": 62246,
            "addon_slug": "local_sambanas2",
        },
        entry_id="test_gateway_host",
    )
    entry.add_to_hass(hass)

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch(
            "custom_components.srat.SRATWebSocketClient",
        ) as mock_ws_cls,
    ):
        mock_ws = AsyncMock()
        mock_ws.register_listener = lambda event, cb: lambda: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.LOADED
    assert mock_ws_cls.call_args.kwargs["host"] == SUPERVISOR_GATEWAY_HOST
    assert mock_ws_cls.call_args.kwargs["addon_slug"] is None


async def test_setup_entry_auto_supervisor_endpoint_resolution(
    hass: HomeAssistant,
) -> None:
    """Test auto supervisor mode resolves runtime host/port via Supervisor API."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data={
            "host": CONF_HOST_AUTO,
            "port": CONF_PORT_AUTO,
            "addon_slug": "local_sambanas2",
        },
        entry_id="test_auto_supervisor_endpoint",
    )
    entry.add_to_hass(hass)

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch(
            "custom_components.srat.resolve_supervisor_addon_endpoint",
            new=AsyncMock(return_value=("172.30.32.1", 64289)),
        ) as mock_resolver,
        patch(
            "custom_components.srat.SRATWebSocketClient",
        ) as mock_ws_cls,
    ):
        mock_ws = AsyncMock()
        mock_ws.register_listener = lambda event, cb: lambda: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.LOADED
    mock_resolver.assert_awaited_once_with(
        hass,
        "local_sambanas2",
        fallback_port=ADDON_API_PORT,
    )
    assert mock_ws_cls.call_args.kwargs["host"] == "172.30.32.1"
    assert mock_ws_cls.call_args.kwargs["port"] == 64289


async def test_setup_entry_migrates_legacy_hassio_entry_to_auto_mode(
    hass: HomeAssistant,
) -> None:
    """Legacy hassio entries with explicit host/port are migrated to auto mode."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data={
            "host": "172.30.32.1",
            "port": 64289,
            "addon_slug": "local_sambanas2",
        },
        entry_id="test_migrate_legacy",
    )
    entry.add_to_hass(hass)

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch.object(hass.config_entries, "async_update_entry") as mock_update_entry,
        patch(
            "custom_components.srat.SRATWebSocketClient",
        ) as mock_ws_cls,
    ):
        mock_ws = AsyncMock()
        mock_ws.register_listener = lambda event, cb: lambda: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.LOADED
    mock_update_entry.assert_called_once()
    updated_data = mock_update_entry.call_args.kwargs["data"]
    assert updated_data["host"] == CONF_HOST_AUTO
    assert updated_data["port"] == CONF_PORT_AUTO
    assert updated_data["addon_slug"] == "local_sambanas2"


async def test_setup_entry_does_not_migrate_manual_remote_entry(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Manual/remote entries without addon slug keep explicit host/port."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_no_migrate_manual",
    )
    entry.add_to_hass(hass)

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch.object(hass.config_entries, "async_update_entry") as mock_update_entry,
        patch(
            "custom_components.srat.SRATWebSocketClient",
        ) as mock_ws_cls,
    ):
        mock_ws = AsyncMock()
        mock_ws.register_listener = lambda event, cb: lambda: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.LOADED
    mock_update_entry.assert_not_called()


async def test_unload_entry(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
) -> None:
    """Test that unloading a config entry disconnects the WS client."""
    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_unload",
    )
    entry.add_to_hass(hass)

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=_mock_session(200),
        ),
        patch(
            "custom_components.srat.SRATWebSocketClient",
        ) as mock_ws_cls,
    ):
        mock_ws = AsyncMock()
        mock_ws.register_listener = lambda event, cb: lambda: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

        await hass.config_entries.async_unload(entry.entry_id)
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.NOT_LOADED
    mock_ws.async_disconnect.assert_called_once()
