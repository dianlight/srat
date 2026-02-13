"""Tests for SRAT sensor entities."""

from __future__ import annotations

from typing import Any
from unittest.mock import AsyncMock

from homeassistant.core import HomeAssistant
from pytest_homeassistant_custom_component.common import MockConfigEntry

from custom_components.srat.const import DOMAIN
from custom_components.srat.coordinator import SRATDataCoordinator
from custom_components.srat.sensor import (
    SRATDiskIOSensor,
    SRATDiskSensor,
    SRATGlobalDiskHealthSensor,
    SRATPartitionHealthSensor,
    SRATPartitionSensor,
    SRATSambaProcessStatusSensor,
    SRATSambaStatusSensor,
    SRATVolumeStatusSensor,
)
from custom_components.srat.websocket_client import SRATWebSocketClient


def _make_coordinator(
    hass: HomeAssistant,
    data: dict[str, Any] | None = None,
) -> SRATDataCoordinator:
    """Create a coordinator with mock WS client and pre-loaded data."""
    ws_client = AsyncMock(spec=SRATWebSocketClient)
    ws_client.register_listener = lambda event, cb: None

    coordinator = SRATDataCoordinator(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        ws_client=ws_client,
    )
    if data is not None:
        coordinator.data = data
    return coordinator


def _make_entry() -> MockConfigEntry:
    """Create a mock config entry."""
    return MockConfigEntry(
        domain=DOMAIN,
        data={"host": "192.168.1.100", "port": 8099},
        entry_id="test_sensor_entry",
    )


# -- SRATSambaStatusSensor --


async def test_samba_status_connected(
    hass: HomeAssistant,
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Test samba status reports 'connected' when sessions exist."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": None,
            "samba_status": mock_heartbeat_data["samba_status"],
            "process_status": None,
            "disk_health": None,
        },
    )
    entry = _make_entry()
    sensor = SRATSambaStatusSensor(coordinator, entry)

    assert sensor.native_value == "connected"
    attrs = sensor.extra_state_attributes
    assert attrs["version"] == "4.18.0"
    assert attrs["session_count"] == 1


async def test_samba_status_idle(hass: HomeAssistant) -> None:
    """Test samba status reports 'idle' when no sessions."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": None,
            "samba_status": {"version": "4.18.0", "sessions": [], "tcons": []},
            "process_status": None,
            "disk_health": None,
        },
    )
    entry = _make_entry()
    sensor = SRATSambaStatusSensor(coordinator, entry)

    assert sensor.native_value == "idle"


async def test_samba_status_unavailable(hass: HomeAssistant) -> None:
    """Test samba status returns None when data not yet received."""
    coordinator = _make_coordinator(hass)
    entry = _make_entry()
    sensor = SRATSambaStatusSensor(coordinator, entry)

    assert sensor.native_value is None


# -- SRATSambaProcessStatusSensor --


async def test_process_status_running(
    hass: HomeAssistant,
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Test process status reports 'running' when all processes are running."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": None,
            "samba_status": None,
            "process_status": mock_heartbeat_data["samba_process_status"],
            "disk_health": None,
        },
    )
    entry = _make_entry()
    sensor = SRATSambaProcessStatusSensor(coordinator, entry)

    assert sensor.native_value == "running"


async def test_process_status_partial(hass: HomeAssistant) -> None:
    """Test process status reports 'partial' when some processes are stopped."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": None,
            "samba_status": None,
            "process_status": {
                "smbd": {"is_running": True, "pid": 1234},
                "nmbd": {"is_running": False},
            },
            "disk_health": None,
        },
    )
    entry = _make_entry()
    sensor = SRATSambaProcessStatusSensor(coordinator, entry)

    assert sensor.native_value == "partial"


async def test_process_status_stopped(hass: HomeAssistant) -> None:
    """Test process status reports 'stopped' when all processes are stopped."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": None,
            "samba_status": None,
            "process_status": {
                "smbd": {"is_running": False},
                "nmbd": {"is_running": False},
            },
            "disk_health": None,
        },
    )
    entry = _make_entry()
    sensor = SRATSambaProcessStatusSensor(coordinator, entry)

    assert sensor.native_value == "stopped"


# -- SRATVolumeStatusSensor --


async def test_volume_status(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Test volume status returns disk count."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": mock_disks_data,
            "samba_status": None,
            "process_status": None,
            "disk_health": None,
        },
    )
    entry = _make_entry()
    sensor = SRATVolumeStatusSensor(coordinator, entry)

    assert sensor.native_value == 1
    attrs = sensor.extra_state_attributes
    assert attrs["disk_count"] == 1
    assert attrs["partition_count"] == 1


async def test_volume_status_unavailable(hass: HomeAssistant) -> None:
    """Test volume status returns None when no disk data."""
    coordinator = _make_coordinator(hass)
    entry = _make_entry()
    sensor = SRATVolumeStatusSensor(coordinator, entry)

    assert sensor.native_value is None


# -- SRATDiskSensor --


async def test_disk_sensor(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Test individual disk sensor."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": mock_disks_data,
            "samba_status": None,
            "process_status": None,
            "disk_health": None,
        },
    )
    entry = _make_entry()
    disk_data = mock_disks_data[0]
    sensor = SRATDiskSensor(coordinator, entry, disk_data)

    assert sensor.native_value == "connected"
    attrs = sensor.extra_state_attributes
    assert attrs["device"] == "/dev/sda"
    assert attrs["model"] == "Samsung SSD 870"
    assert attrs["partition_count"] == 1


# -- SRATPartitionSensor --


async def test_partition_sensor_shared(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Test partition sensor reports 'shared' when shares exist."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": mock_disks_data,
            "samba_status": None,
            "process_status": None,
            "disk_health": None,
        },
    )
    entry = _make_entry()
    disk_data = mock_disks_data[0]
    part_data = disk_data["partitions"][0]
    sensor = SRATPartitionSensor(coordinator, entry, part_data, disk_data)

    assert sensor.native_value == "shared"


# -- SRATGlobalDiskHealthSensor --


async def test_global_disk_health(
    hass: HomeAssistant,
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Test global disk health returns total IOPS."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": None,
            "samba_status": None,
            "process_status": None,
            "disk_health": mock_heartbeat_data["disk_health"],
        },
    )
    entry = _make_entry()
    sensor = SRATGlobalDiskHealthSensor(coordinator, entry)

    assert sensor.native_value == 150.5


# -- SRATDiskIOSensor --


async def test_disk_io_sensor(
    hass: HomeAssistant,
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Test per-disk IO sensor returns total IOPS."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": None,
            "samba_status": None,
            "process_status": None,
            "disk_health": mock_heartbeat_data["disk_health"],
        },
    )
    entry = _make_entry()
    sensor = SRATDiskIOSensor(
        coordinator,
        entry,
        "sda",
        mock_heartbeat_data["disk_health"]["disk_io"]["sda"],
    )

    assert sensor.native_value == 150.5
    attrs = sensor.extra_state_attributes
    assert attrs["smart_temperature"] == 35


# -- SRATPartitionHealthSensor --


async def test_partition_health_sensor(
    hass: HomeAssistant,
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Test partition health sensor returns free space."""
    coordinator = _make_coordinator(
        hass,
        {
            "disks": None,
            "samba_status": None,
            "process_status": None,
            "disk_health": mock_heartbeat_data["disk_health"],
        },
    )
    entry = _make_entry()
    sensor = SRATPartitionHealthSensor(
        coordinator,
        entry,
        "/dev/sda1",
        mock_heartbeat_data["disk_health"]["partition_health"]["/dev/sda1"],
    )

    assert sensor.native_value == 250000000000
    attrs = sensor.extra_state_attributes
    assert attrs["fstype"] == "ext4"
    assert attrs["usage_percent"] == 50.0
