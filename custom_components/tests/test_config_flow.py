"""Tests for the SRAT config flow."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

import aiohttp
from homeassistant.components.hassio import HassioServiceInfo
from homeassistant.config_entries import SOURCE_HASSIO, SOURCE_USER
from homeassistant.core import HomeAssistant
from homeassistant.data_entry_flow import FlowResultType

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


async def test_user_flow_success(
    hass: HomeAssistant,
    mock_setup_entry: AsyncMock,
) -> None:
    """Test successful manual configuration."""
    result = await hass.config_entries.flow.async_init(
        DOMAIN, context={"source": SOURCE_USER}
    )
    assert result["type"] is FlowResultType.FORM
    assert result["step_id"] == "user"

    with patch(
        "custom_components.srat.config_flow.async_get_clientsession",
        return_value=_mock_session(200),
    ):
        result = await hass.config_entries.flow.async_configure(
            result["flow_id"],
            user_input={"host": "192.168.1.100", "port": 8099},
        )

    assert result["type"] is FlowResultType.CREATE_ENTRY
    assert result["title"] == "SRAT (192.168.1.100:8099)"
    assert result["data"] == {"host": "192.168.1.100", "port": 8099}


async def test_user_flow_cannot_connect(
    hass: HomeAssistant,
) -> None:
    """Test error when connection fails."""
    result = await hass.config_entries.flow.async_init(
        DOMAIN, context={"source": SOURCE_USER}
    )

    with patch(
        "custom_components.srat.config_flow.async_get_clientsession",
        return_value=_mock_session(503),
    ):
        result = await hass.config_entries.flow.async_configure(
            result["flow_id"],
            user_input={"host": "192.168.1.100", "port": 8099},
        )

    assert result["type"] is FlowResultType.FORM
    assert result["errors"] == {"base": "cannot_connect"}


async def test_user_flow_connection_error(
    hass: HomeAssistant,
) -> None:
    """Test error when aiohttp raises ClientError."""
    result = await hass.config_entries.flow.async_init(
        DOMAIN, context={"source": SOURCE_USER}
    )

    mock_ctx = AsyncMock()
    mock_ctx.__aenter__ = AsyncMock(side_effect=aiohttp.ClientError())
    mock_ctx.__aexit__ = AsyncMock(return_value=False)

    session = MagicMock(spec=aiohttp.ClientSession)
    session.get = MagicMock(return_value=mock_ctx)

    with patch(
        "custom_components.srat.config_flow.async_get_clientsession",
        return_value=session,
    ):
        result = await hass.config_entries.flow.async_configure(
            result["flow_id"],
            user_input={"host": "192.168.1.100", "port": 8099},
        )

    assert result["type"] is FlowResultType.FORM
    assert result["errors"] == {"base": "cannot_connect"}


async def test_user_flow_shows_form(hass: HomeAssistant) -> None:
    """Test that the user step shows a form when no input is given."""
    result = await hass.config_entries.flow.async_init(
        DOMAIN, context={"source": SOURCE_USER}
    )
    assert result["type"] is FlowResultType.FORM
    assert result["step_id"] == "user"
    assert result["errors"] == {}


async def test_hassio_discovery(
    hass: HomeAssistant,
    mock_setup_entry: AsyncMock,
) -> None:
    """Test Supervisor add-on auto-discovery."""
    discovery_info = HassioServiceInfo(
        config={"host": "core-local-sambanas2", "port": 8099},
        name="SambaNas2",
        slug="local_sambanas2",
        uuid="test-uuid-1234",
    )

    result = await hass.config_entries.flow.async_init(
        DOMAIN,
        context={"source": SOURCE_HASSIO},
        data=discovery_info,
    )

    assert result["type"] is FlowResultType.FORM
    assert result["step_id"] == "hassio_confirm"

    result = await hass.config_entries.flow.async_configure(
        result["flow_id"],
        user_input={},
    )

    assert result["type"] is FlowResultType.CREATE_ENTRY
    assert result["title"] == "SRAT"
    assert result["data"]["host"] == "core-local-sambanas2"
    assert result["data"]["port"] == 8099


async def test_hassio_discovery_rejects_unknown_slug(
    hass: HomeAssistant,
) -> None:
    """Test that unknown addon slugs are rejected."""
    discovery_info = HassioServiceInfo(
        config={},
        name="Unknown Addon",
        slug="unknown_addon",
        uuid="test-uuid-5678",
    )

    result = await hass.config_entries.flow.async_init(
        DOMAIN,
        context={"source": SOURCE_HASSIO},
        data=discovery_info,
    )

    assert result["type"] is FlowResultType.ABORT
    assert result["reason"] == "not_srat_addon"
