"""Tests for the SRAT data coordinator."""

from __future__ import annotations

from typing import Any
from unittest.mock import AsyncMock

from homeassistant.core import HomeAssistant

from custom_components.srat.coordinator import SRATDataCoordinator
from custom_components.srat.websocket_client import SRATWebSocketClient


def _make_coordinator(
    hass: HomeAssistant,
) -> tuple[SRATDataCoordinator, dict[str, Any]]:
    """Create a coordinator with a mock WS client and capture registered listeners."""
    listeners: dict[str, Any] = {}
    ws_client = AsyncMock(spec=SRATWebSocketClient)
    ws_client.register_listener = lambda event, cb: listeners.update({event: cb})

    coordinator = SRATDataCoordinator(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        ws_client=ws_client,
    )
    return coordinator, listeners


async def test_initial_data_is_none(hass: HomeAssistant) -> None:
    """Test that initial data keys are None (sensors report unavailable)."""
    coordinator, _ = _make_coordinator(hass)

    assert coordinator.data is not None
    assert coordinator.data["disks"] is None
    assert coordinator.data["samba_status"] is None
    assert coordinator.data["process_status"] is None
    assert coordinator.data["disk_health"] is None


async def test_volumes_event(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Test that ``volumes`` event updates disk data."""
    coordinator, listeners = _make_coordinator(hass)
    assert "volumes" in listeners

    listeners["volumes"](mock_disks_data)

    assert coordinator.data["disks"] == mock_disks_data
    assert len(coordinator.data["disks"]) == 1


async def test_heartbeat_event(
    hass: HomeAssistant,
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Test that ``heartbeat`` event updates samba and health data."""
    coordinator, listeners = _make_coordinator(hass)
    assert "heartbeat" in listeners

    listeners["heartbeat"](mock_heartbeat_data)

    assert coordinator.data["samba_status"] is not None
    assert coordinator.data["samba_status"]["version"] == "4.18.0"
    assert coordinator.data["process_status"] is not None
    assert coordinator.data["disk_health"] is not None


async def test_heartbeat_ignores_non_dict(hass: HomeAssistant) -> None:
    """Test that non-dict heartbeat payloads are ignored."""
    coordinator, listeners = _make_coordinator(hass)

    listeners["heartbeat"]("not a dict")

    assert coordinator.data["samba_status"] is None
    assert coordinator.data["process_status"] is None


async def test_volumes_non_list_sets_none(hass: HomeAssistant) -> None:
    """Test that non-list volumes payload sets disks to None."""
    coordinator, listeners = _make_coordinator(hass)

    listeners["volumes"]("not a list")

    assert coordinator.data["disks"] is None


async def test_multiple_events_update_data(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Test that multiple events accumulate data correctly."""
    coordinator, listeners = _make_coordinator(hass)

    listeners["volumes"](mock_disks_data)
    listeners["heartbeat"](mock_heartbeat_data)

    assert coordinator.data["disks"] is not None
    assert coordinator.data["samba_status"] is not None
    assert coordinator.data["process_status"] is not None
    assert coordinator.data["disk_health"] is not None
