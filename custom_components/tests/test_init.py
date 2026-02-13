"""Tests for SRAT integration setup and teardown."""

from __future__ import annotations

from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import aiohttp
from homeassistant.config_entries import ConfigEntryState
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
        mock_ws.register_listener = lambda event, cb: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.LOADED


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
        mock_ws.register_listener = lambda event, cb: None
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()
        assert entry.state is ConfigEntryState.LOADED

        await hass.config_entries.async_unload(entry.entry_id)
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.NOT_LOADED
    mock_ws.async_disconnect.assert_called_once()
