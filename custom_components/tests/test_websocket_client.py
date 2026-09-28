"""Tests for the SRAT WebSocket client handshake."""

from __future__ import annotations

import asyncio
from collections.abc import Callable
from types import TracebackType
from typing import Any
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

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> bool:
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

    assert (
        session.ws_connect.call_args.kwargs["headers"] == homeassistant_auth_headers()
    )

    ws.send_json.assert_awaited_once_with(
        {
            "type": "helo",
            "component": "srat",
            "version": "2026.03.1",
            "ha_version": None,
            "entry_id": None,
            "capabilities": [],
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

    expected_payload: dict[str, Any] = {
        "type": "helo",
        "component": "srat",
        "version": "2026.03.1",
        "ha_version": None,
        "entry_id": None,
        "capabilities": [],
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
            "ha_version": None,
            "entry_id": None,
            "capabilities": [],
            "addon_slug": "local_sambanas2",
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
            "event": "repair_lifecycle",
            "data": {
                "repair_id": "disk_space_low",
                "command_id": "cmd-1",
                "status": "created",
                "details": {"attempt": 1},
            },
        }
    )


async def test_async_ensure_running_restarts_listener_when_task_stopped(
    hass: HomeAssistant,
) -> None:
    """Ensure listener is restarted if task unexpectedly stopped."""
    client = SRATWebSocketClient(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        integration_version="2026.03.1",
    )
    client._should_reconnect = True

    task = hass.async_create_background_task(asyncio.sleep(0), "done_task")
    await task
    client._task = task

    with patch.object(client, "_start_listener_task") as mock_start:
        await client.async_ensure_running()

    mock_start.assert_called_once()


async def test_listen_loop_reconnects_immediately_after_clean_drop(
    hass: HomeAssistant,
) -> None:
    """A cleanly closed connection must trigger an immediate reconnect, no sleep."""
    client = SRATWebSocketClient(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        reconnect_interval=5,
        integration_version="2026.03.1",
    )
    client._should_reconnect = True

    first_ws = AsyncMock(spec=aiohttp.ClientWebSocketResponse)
    first_ws.__aiter__.return_value = []  # clean close, no messages

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

    with (
        patch(
            "custom_components.srat.websocket_client.async_get_clientsession",
            return_value=session,
        ),
        patch("asyncio.sleep") as mock_sleep,
    ):
        await client._listen_loop()

    mock_sleep.assert_not_called()


async def test_listen_loop_delays_retry_after_connection_failure(
    hass: HomeAssistant,
) -> None:
    """A failed connection attempt must back off by reconnect_interval before retry."""
    client = SRATWebSocketClient(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        reconnect_interval=5,
        integration_version="2026.03.1",
    )
    client._should_reconnect = True

    second_ws = AsyncMock(spec=aiohttp.ClientWebSocketResponse)
    second_ws.__aiter__.return_value = []

    session = MagicMock(spec=aiohttp.ClientSession)
    session.ws_connect = MagicMock(
        side_effect=[
            aiohttp.ClientConnectionError("connection refused"),
            _WebSocketContextManager(
                second_ws,
                lambda: setattr(client, "_should_reconnect", False),
            ),
        ]
    )

    with (
        patch(
            "custom_components.srat.websocket_client.async_get_clientsession",
            return_value=session,
        ),
        patch("asyncio.sleep") as mock_sleep,
    ):
        await client._listen_loop()

    mock_sleep.assert_called_once_with(5)


async def test_listen_loop_calls_endpoint_resolver_before_each_reconnect(
    hass: HomeAssistant,
) -> None:
    """Endpoint resolver must be invoked before each reconnect attempt."""
    resolve_call_count = 0
    resolved_ports = [3001, 3002]

    async def _resolver() -> tuple[str, int]:
        nonlocal resolve_call_count
        port = resolved_ports[resolve_call_count]
        resolve_call_count += 1
        return "172.30.32.1", port

    client = SRATWebSocketClient(
        hass=hass,
        host="172.30.32.1",
        port=3000,
        reconnect_interval=0,
        integration_version="2026.03.1",
        endpoint_resolver=_resolver,
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

    captured_urls: list[str] = []

    original_ws_connect = session.ws_connect

    def _capturing_ws_connect(url: str, **kwargs: Any) -> Any:
        captured_urls.append(url)
        return original_ws_connect(url, **kwargs)

    session.ws_connect = MagicMock(side_effect=_capturing_ws_connect)

    with patch(
        "custom_components.srat.websocket_client.async_get_clientsession",
        return_value=session,
    ):
        await client._listen_loop()

    assert resolve_call_count == 2
    assert captured_urls[0] == "ws://172.30.32.1:3001/ws"
    assert captured_urls[1] == "ws://172.30.32.1:3002/ws"


def _make_client(hass: HomeAssistant) -> SRATWebSocketClient:
    """Create a plain WebSocket client for message-level tests."""
    return SRATWebSocketClient(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        integration_version="2026.03.1",
    )


async def test_parse_ws_message_dispatches_to_listener(
    hass: HomeAssistant,
) -> None:
    """A valid SSE frame must reach the registered listener as parsed JSON."""
    client = _make_client(hass)
    received: list[Any] = []
    client.register_listener("volumes", received.append)

    client._parse_ws_message('id: 1\nevent: volumes\ndata: {"a": 1}')

    assert received == [{"a": 1}]


async def test_parse_ws_message_malformed_frame_ignored(
    hass: HomeAssistant,
) -> None:
    """A malformed frame must be dropped without calling listeners."""
    client = _make_client(hass)
    received: list[Any] = []
    client.register_listener("volumes", received.append)

    client._parse_ws_message("not-an-sse-frame")

    assert received == []


async def test_parse_ws_message_bad_json_ignored(hass: HomeAssistant) -> None:
    """A frame with non-JSON data must be dropped without calling listeners."""
    client = _make_client(hass)
    received: list[Any] = []
    client.register_listener("volumes", received.append)

    client._parse_ws_message("id: 1\nevent: volumes\ndata: {oops")

    assert received == []


async def test_parse_ws_message_listener_exception_isolated(
    hass: HomeAssistant,
) -> None:
    """A raising listener must not prevent sibling listeners from running."""

    def _boom(payload: Any) -> None:
        raise RuntimeError("listener boom")

    client = _make_client(hass)
    received: list[Any] = []
    client.register_listener("volumes", _boom)
    client.register_listener("volumes", received.append)

    client._parse_ws_message('id: 1\nevent: volumes\ndata: {"a": 1}')

    assert received == [{"a": 1}]


async def test_remove_listener_twice_is_safe(hass: HomeAssistant) -> None:
    """Unregistering the same listener twice must not raise."""
    client = _make_client(hass)
    remove = client.register_listener("volumes", lambda payload: None)

    remove()
    remove()

    assert "volumes" not in client._listeners


async def test_on_listener_task_done_skipped_when_reconnect_disabled(
    hass: HomeAssistant,
    caplog: Any,
) -> None:
    """A finished task must be ignored silently when reconnect is disabled."""
    client = _make_client(hass)
    client._should_reconnect = False
    task = MagicMock()
    task.cancelled.return_value = False
    task.exception.return_value = RuntimeError("boom")

    client._on_listener_task_done(task)

    task.cancelled.assert_not_called()
    assert "watchdog will restart" not in caplog.text


async def test_on_listener_task_done_ignores_cancelled_task(
    hass: HomeAssistant,
    caplog: Any,
) -> None:
    """A cancelled listener task must be ignored silently."""
    client = _make_client(hass)
    client._should_reconnect = True
    task = MagicMock()
    task.cancelled.return_value = True

    client._on_listener_task_done(task)

    task.exception.assert_not_called()
    assert "watchdog will restart" not in caplog.text


async def test_on_listener_task_done_warns_on_crash(
    hass: HomeAssistant,
    caplog: Any,
) -> None:
    """A crashed listener task must log that the watchdog will restart it."""
    client = _make_client(hass)
    client._should_reconnect = True
    task = MagicMock()
    task.cancelled.return_value = False
    task.exception.return_value = RuntimeError("boom")

    with caplog.at_level("WARNING", logger="custom_components.srat.websocket_client"):
        client._on_listener_task_done(task)

    assert "watchdog will restart" in caplog.text


async def test_on_listener_task_done_warns_on_clean_stop(
    hass: HomeAssistant,
    caplog: Any,
) -> None:
    """A cleanly stopped listener task must log that the watchdog will restart it."""
    client = _make_client(hass)
    client._should_reconnect = True
    task = MagicMock()
    task.cancelled.return_value = False
    task.exception.return_value = None

    with caplog.at_level("WARNING", logger="custom_components.srat.websocket_client"):
        client._on_listener_task_done(task)

    assert "stopped unexpectedly" in caplog.text


async def test_listen_loop_cancel_during_retry_sleep_stops(
    hass: HomeAssistant,
) -> None:
    """Cancellation during the reconnect sleep must stop the loop."""
    client = _make_client(hass)
    client._reconnect_interval = 5
    client._should_reconnect = True

    session = MagicMock(spec=aiohttp.ClientSession)
    session.ws_connect = MagicMock(
        side_effect=aiohttp.ClientConnectionError("connection refused")
    )

    with (
        patch(
            "custom_components.srat.websocket_client.async_get_clientsession",
            return_value=session,
        ),
        patch("asyncio.sleep", side_effect=asyncio.CancelledError()),
    ):
        await client._listen_loop()

    assert client._should_reconnect is False


async def test_listen_loop_cancel_during_flap_sleep_stops(
    hass: HomeAssistant,
) -> None:
    """Cancellation during the flap backoff sleep must stop the loop."""
    client = _make_client(hass)
    client._reconnect_interval = 5
    client._should_reconnect = True

    ws = AsyncMock(spec=aiohttp.ClientWebSocketResponse)
    error_msg = MagicMock()
    error_msg.type = aiohttp.WSMsgType.ERROR
    ws.__aiter__.return_value = [error_msg]
    ws.exception = MagicMock(return_value="boom")

    session = MagicMock(spec=aiohttp.ClientSession)
    session.ws_connect = MagicMock(
        return_value=_WebSocketContextManager(ws, lambda: None)
    )

    with (
        patch(
            "custom_components.srat.websocket_client.async_get_clientsession",
            return_value=session,
        ),
        patch("asyncio.sleep", side_effect=asyncio.CancelledError()),
    ):
        await client._listen_loop()

    assert client._should_reconnect is False
    assert session.ws_connect.call_count == 1
