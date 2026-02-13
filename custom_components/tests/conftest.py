"""Shared fixtures for SRAT integration tests."""

from __future__ import annotations

from collections.abc import Generator
from typing import Any
from unittest.mock import AsyncMock, patch

from homeassistant.const import CONF_HOST, CONF_PORT
import pytest


@pytest.fixture(autouse=True)
def auto_enable_custom_integrations(
    enable_custom_integrations: None,
) -> None:
    """Enable custom integrations in all tests."""


@pytest.fixture
def mock_config_entry_data() -> dict[str, Any]:
    """Return mock config entry data."""
    return {
        CONF_HOST: "192.168.1.100",
        CONF_PORT: 8099,
    }


@pytest.fixture
def mock_health_response() -> dict[str, Any]:
    """Return a mock /api/health response."""
    return {"status": "ok"}


@pytest.fixture
def mock_disks_data() -> list[dict[str, Any]]:
    """Return mock disk data (from ``volumes`` WS event)."""
    return [
        {
            "id": "disk-001",
            "device": "/dev/sda",
            "model": "Samsung SSD 870",
            "vendor": "Samsung",
            "serial": "S1234567890",
            "size": 500107862016,
            "connectionBus": "usb",
            "removable": False,
            "partitions": [
                {
                    "id": "part-001",
                    "device": "/dev/sda1",
                    "name": "data",
                    "size": 499999997952,
                    "system": False,
                    "mounts": ["/mnt/data"],
                    "shares": ["share1"],
                },
            ],
        },
    ]


@pytest.fixture
def mock_heartbeat_data() -> dict[str, Any]:
    """Return mock heartbeat data (``HealthPing`` from ``heartbeat`` WS event)."""
    return {
        "alive": True,
        "aliveTime": 1700000000,
        "samba_status": {
            "version": "4.18.0",
            "sessions": [{"uid": 1000, "username": "user1"}],
            "tcons": [{"service": "share1"}],
        },
        "samba_process_status": {
            "smbd": {
                "is_running": True,
                "pid": 1234,
                "cpu_percent": 1.5,
                "memory_percent": 0.8,
            },
            "nmbd": {
                "is_running": True,
                "pid": 1235,
                "cpu_percent": 0.2,
                "memory_percent": 0.3,
            },
        },
        "disk_health": {
            "global": {
                "total_iops": 150.5,
                "total_read_iops": 100.0,
                "total_write_iops": 50.5,
                "avg_read_latency_ms": 0.5,
                "avg_write_latency_ms": 1.2,
            },
            "disk_io": {
                "sda": {
                    "read_iops": 100.0,
                    "write_iops": 50.5,
                    "read_latency_ms": 0.5,
                    "write_latency_ms": 1.2,
                    "smart_temperature": 35,
                    "smart_power_on_hours": 5000,
                    "smart_power_cycle_count": 100,
                },
            },
            "partition_health": {
                "/dev/sda1": {
                    "mount_point": "/mnt/data",
                    "fstype": "ext4",
                    "total_space_bytes": 499999997952,
                    "free_space_bytes": 250000000000,
                    "usage_percent": 50.0,
                    "fsck_needed": False,
                    "fsck_supported": True,
                    "disk_name": "sda",
                },
            },
        },
    }


@pytest.fixture
def mock_setup_entry() -> Generator[AsyncMock]:
    """Mock a successful setup entry."""
    with patch(
        "custom_components.srat.async_setup_entry",
        return_value=True,
    ) as mock_setup:
        yield mock_setup
