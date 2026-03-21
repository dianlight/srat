"""Tests for the SRAT WebSocket client handshake."""

from __future__ import annotations

from collections.abc import Callable
from unittest.mock import AsyncMock, MagicMock, patch

import aiohttp
from homeassistant.core import HomeAssistant

from custom_components.srat.connection import homeassistant_auth_headers
from custom_components.srat.websocket_client import SRATWebSocketClient


class _WebSocketContextManager:
    """Async context manager wrapper for a mocked aiohttp WebSocket."""

    def __init__(self, ws: AsyncMock, on_exit: Callable[[], None]) -> None:
        """Store the WebSocket mock and exit hook."""
        self._ws = ws
        self._on_exit = on_exit

    async def __aenter__(self) -> AsyncMock:
        """Return the mocked WebSocket response."""
        return self._ws

    async def __aexit__(self, exc_type, exc, tb) -> bool:
        """Trigger the exit hook and propagate exceptions."""
        self._on_exit()
        return False


async def test_listen_loop_sends_helo_on_connect(hass: HomeAssistant) -> None:
    """Test that a successful connection sends the initial helo payload."""
    client = SRATWebSocketClient(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        integration_version="2026.03.1",
    )
    client._should_reconnect = True

    ws = AsyncMock(spec=aiohttp.ClientWebSocketResponse)
    ws.__aiter__.return_value = []

    session = MagicMock(spec=aiohttp.ClientSession)
    session.ws_connect = MagicMock(
        return_value=_WebSocketContextManager(
            ws,
            lambda: setattr(client, "_should_reconnect", False),
        )
    )

    with patch(
        "custom_components.srat.websocket_client.async_get_clientsession",
        return_value=session,
    ):
        await client._listen_loop()

    assert session.ws_connect.call_args.kwargs["headers"] == homeassistant_auth_headers()

    ws.send_json.assert_awaited_once_with(
        {
            "type": "helo",
            "component": "srat",
            "version": "2026.03.1",
        }
    )


async def test_listen_loop_resends_helo_after_reconnect(
    hass: HomeAssistant,
) -> None:
    """Test that the client sends helo again after a reconnect."""
    client = SRATWebSocketClient(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        reconnect_interval=0,
        integration_version="2026.03.1",
    )
    client._should_reconnect = True

    first_ws = AsyncMock(spec=aiohttp.ClientWebSocketResponse)
    first_ws.__aiter__.return_value = []

    second_ws = AsyncMock(spec=aiohttp.ClientWebSocketResponse)
    second_ws.__aiter__.return_value = []

    session = MagicMock(spec=aiohttp.ClientSession)
    session.ws_connect = MagicMock(
        side_effect=[
            _WebSocketContextManager(first_ws, lambda: None),
            _WebSocketContextManager(
                second_ws,
                lambda: setattr(client, "_should_reconnect", False),
            ),
        ]
    )

    with patch(
        "custom_components.srat.websocket_client.async_get_clientsession",
        return_value=session,
    ):
        await client._listen_loop()

    expected_payload = {
        "type": "helo",
        "component": "srat",
        "version": "2026.03.1",
    }
    first_ws.send_json.assert_awaited_once_with(expected_payload)
    second_ws.send_json.assert_awaited_once_with(expected_payload)


async def test_listen_loop_prefers_supervisor_gateway_host(
    hass: HomeAssistant,
) -> None:
    """Test that Supervisor add-on connections try the gateway host first."""
    client = SRATWebSocketClient(
        hass=hass,
        host="local-sambanas2",
        port=62246,
        reconnect_interval=0,
        integration_version="2026.03.1",
        addon_slug="local_sambanas2",
    )
    client._should_reconnect = True

    ws = AsyncMock(spec=aiohttp.ClientWebSocketResponse)
    ws.__aiter__.return_value = []

    session = MagicMock(spec=aiohttp.ClientSession)
    session.ws_connect = MagicMock(
        side_effect=[
            aiohttp.ClientConnectionError("gateway failed"),
            _WebSocketContextManager(
                ws,
                lambda: setattr(client, "_should_reconnect", False),
            ),
        ]
    )

    with patch(
        "custom_components.srat.websocket_client.async_get_clientsession",
        return_value=session,
    ):
        await client._listen_loop()

    assert session.ws_connect.call_args_list[0].args[0] == "ws://172.30.32.1:62246/ws"
    assert (
        session.ws_connect.call_args_list[1].args[0] == "ws://local-sambanas2:62246/ws"
    )
    ws.send_json.assert_awaited_once_with(
        {
            "type": "helo",
            "component": "srat",
            "version": "2026.03.1",
        }
    )


async def test_send_repair_lifecycle_event_when_connected(
    hass: HomeAssistant,
) -> None:
    """Test sending repair lifecycle payload over active websocket connection."""
    client = SRATWebSocketClient(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        integration_version="2026.03.1",
    )

    ws = AsyncMock(spec=aiohttp.ClientWebSocketResponse)
    client._connected = True
    client._ws = ws

    await client.async_send_repair_lifecycle_event(
        repair_id="disk_space_low",
        command_id="cmd-1",
        status="created",
        details={"attempt": 1},
    )

    ws.send_json.assert_awaited_once_with(
        {
            "type": "repair_lifecycle",
            "repair_id": "disk_space_low",
            "command_id": "cmd-1",
            "status": "created",
            "details": {"attempt": 1},
        }
    )
