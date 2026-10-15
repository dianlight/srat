"""Tests for SRAT sensor entities."""

from __future__ import annotations

from collections.abc import Iterable
from types import SimpleNamespace
from typing import Any
from unittest.mock import AsyncMock

from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity import Entity
from homeassistant.helpers.entity_platform import AddEntitiesCallback
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
    VolumeRepository,
    async_setup_entry,
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


def _make_listening_coordinator(
    hass: HomeAssistant,
) -> tuple[SRATDataCoordinator, dict[str, Any]]:
    """Create a coordinator that captures registered WS listeners."""
    listeners: dict[str, Any] = {}
    ws_client = AsyncMock(spec=SRATWebSocketClient)

    def _register(event: str, cb: Any) -> Any:
        listeners[event] = cb
        return lambda: None

    ws_client.register_listener = _register
    coordinator = SRATDataCoordinator(
        hass=hass,
        host="192.168.1.100",
        port=8099,
        ws_client=ws_client,
    )
    return coordinator, listeners


# -- async_setup_entry dynamic entity creation (issue #1263) --


async def test_setup_defers_disk_entities_until_volumes_event(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Disk/partition sensors are created when the first volumes event arrives."""
    coordinator, listeners = _make_listening_coordinator(hass)
    entry = _make_entry()
    entry.runtime_data = SimpleNamespace(coordinator=coordinator)

    added: list[Entity] = []

    def _add(entities: Iterable[Entity], update_before_add: bool = False) -> None:
        added.extend(entities)

    _add_cb: AddEntitiesCallback = _add  # type: ignore[assignment]

    await async_setup_entry(hass, entry, _add_cb)

    # Only the 4 base sensors exist before any volumes event.
    assert len(added) == 4
    assert all(not isinstance(e, (SRATDiskSensor, SRATPartitionSensor)) for e in added)

    listeners["volumes"](mock_disks_data)

    assert len(added) == 6
    assert isinstance(added[4], SRATDiskSensor)
    assert isinstance(added[5], SRATPartitionSensor)


async def test_setup_creates_disk_entities_immediately_when_data_present(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Disk/partition sensors are created at setup when disks data already exists."""
    coordinator, _ = _make_listening_coordinator(hass)
    coordinator.data["disks"] = mock_disks_data
    entry = _make_entry()
    entry.runtime_data = SimpleNamespace(coordinator=coordinator)

    added: list[Entity] = []

    def _add(entities: Iterable[Entity], update_before_add: bool = False) -> None:
        added.extend(entities)

    _add_cb: AddEntitiesCallback = _add  # type: ignore[assignment]

    await async_setup_entry(hass, entry, _add_cb)

    assert len(added) == 6
    assert isinstance(added[4], SRATDiskSensor)
    assert isinstance(added[5], SRATPartitionSensor)


async def test_setup_defers_health_entities_until_heartbeat(
    hass: HomeAssistant,
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Disk IO / partition health sensors are created when disk_health arrives."""
    coordinator, listeners = _make_listening_coordinator(hass)
    entry = _make_entry()
    entry.runtime_data = SimpleNamespace(coordinator=coordinator)

    added: list[Entity] = []

    def _add(entities: Iterable[Entity], update_before_add: bool = False) -> None:
        added.extend(entities)

    _add_cb: AddEntitiesCallback = _add  # type: ignore[assignment]

    await async_setup_entry(hass, entry, _add_cb)

    assert len(added) == 4

    listeners["heartbeat"](mock_heartbeat_data)

    assert len(added) == 6
    assert isinstance(added[4], SRATDiskIOSensor)
    assert isinstance(added[5], SRATPartitionHealthSensor)


async def test_setup_does_not_duplicate_entities_on_subsequent_events(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Repeated volumes events do not create duplicate entities."""
    coordinator, listeners = _make_listening_coordinator(hass)
    entry = _make_entry()
    entry.runtime_data = SimpleNamespace(coordinator=coordinator)

    added: list[Entity] = []

    def _add(entities: Iterable[Entity], update_before_add: bool = False) -> None:
        added.extend(entities)

    _add_cb: AddEntitiesCallback = _add  # type: ignore[assignment]

    await async_setup_entry(hass, entry, _add_cb)
    listeners["volumes"](mock_disks_data)
    listeners["volumes"](mock_disks_data)

    assert len(added) == 6


# -- End-to-end: full integration setup (issue #1263 repro) --


async def test_full_setup_creates_disk_entities_after_volumes_event(
    hass: HomeAssistant,
    mock_config_entry_data: dict[str, Any],
    mock_disks_data: list[dict[str, Any]],
    mock_heartbeat_data: dict[str, Any],
) -> None:
    """Full integration setup: disk entities materialize after a volumes event.

    Mirrors the issue #1263 repro: fresh backend with disks, integration set up,
    no mount/unmount activity. Before the fix only the 4 base sensors exist.
    """
    from unittest.mock import MagicMock, patch

    import aiohttp
    from homeassistant.config_entries import ConfigEntryState
    from homeassistant.helpers import entity_registry as er
    from homeassistant.setup import async_setup_component

    entry = MockConfigEntry(
        domain=DOMAIN,
        data=mock_config_entry_data,
        entry_id="test_e2e_volumes",
    )
    entry.add_to_hass(hass)

    listeners: dict[str, Any] = {}

    mock_resp = AsyncMock()
    mock_resp.status = 200
    mock_ctx = AsyncMock()
    mock_ctx.__aenter__ = AsyncMock(return_value=mock_resp)
    mock_ctx.__aexit__ = AsyncMock(return_value=False)
    mock_session = MagicMock(spec=aiohttp.ClientSession)
    mock_session.get = MagicMock(return_value=mock_ctx)

    with (
        patch(
            "custom_components.srat.async_get_clientsession",
            return_value=mock_session,
        ),
        patch("custom_components.srat.SRATWebSocketClient") as mock_ws_cls,
    ):
        mock_ws = AsyncMock()

        def _register(event: str, cb: Any) -> Any:
            listeners[event] = cb
            return lambda: None

        mock_ws.register_listener = _register
        mock_ws.async_connect = AsyncMock()
        mock_ws.async_disconnect = AsyncMock()
        mock_ws_cls.return_value = mock_ws

        assert await async_setup_component(hass, DOMAIN, {})
        await hass.async_block_till_done()

    assert entry.state is ConfigEntryState.LOADED

    registry = er.async_get(hass)
    unique_ids = {
        e.unique_id for e in er.async_entries_for_config_entry(registry, entry.entry_id)
    }

    # Only the 4 base sensors before any volumes event.
    assert unique_ids == {
        f"{entry.entry_id}_samba_status",
        f"{entry.entry_id}_samba_process_status",
        f"{entry.entry_id}_volume_status",
        f"{entry.entry_id}_global_disk_health",
    }

    # Fire the volumes event (as the backend would on a mount/unmount).
    listeners["volumes"](mock_disks_data)
    await hass.async_block_till_done()

    unique_ids = {
        e.unique_id for e in er.async_entries_for_config_entry(registry, entry.entry_id)
    }
    assert f"{entry.entry_id}_disk_disk_001" in unique_ids
    assert f"{entry.entry_id}_partition_part_001" in unique_ids

    # Fire the heartbeat (disk_health with disk_io + partition_health).
    listeners["heartbeat"](mock_heartbeat_data)
    await hass.async_block_till_done()

    unique_ids = {
        e.unique_id for e in er.async_entries_for_config_entry(registry, entry.entry_id)
    }
    assert f"{entry.entry_id}_disk_io_sda" in unique_ids
    assert f"{entry.entry_id}_partition_health_dev_sda1" in unique_ids


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
    """Test partition sensor reports 'shared' when an enabled share exists."""
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
    attrs = sensor.extra_state_attributes
    assert attrs["device"] == "/dev/sda1"
    assert attrs["mount_count"] == 1
    assert attrs["share_count"] == 1


async def test_partition_sensor_mounted_only(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Test partition sensor reports 'mounted' when no share but is_mounted."""
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
    # Remove the share but keep the mount
    part_data["mount_point_data"]["/mnt/data"]["share"] = None
    sensor = SRATPartitionSensor(coordinator, entry, part_data, disk_data)

    assert sensor.native_value == "mounted"
    attrs = sensor.extra_state_attributes
    assert attrs["mount_count"] == 1
    assert attrs["share_count"] == 0


async def test_partition_sensor_unmounted(
    hass: HomeAssistant,
    mock_disks_data: list[dict[str, Any]],
) -> None:
    """Test partition sensor reports 'unmounted' when no mounts."""
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
    part_data["mount_point_data"] = {}
    sensor = SRATPartitionSensor(coordinator, entry, part_data, disk_data)

    assert sensor.native_value == "unmounted"
    attrs = sensor.extra_state_attributes
    assert attrs["mount_count"] == 0
    assert attrs["share_count"] == 0


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


# -- VolumeRepository --


def test_find_partition_in_list_payload() -> None:
    """Partitions serialized as a list are found by id."""
    disks = [{"id": "d1", "partitions": [{"id": "p1", "device_path": "sda1"}]}]
    found = VolumeRepository.find_partition(disks, "p1")
    assert found is not None
    assert found["device_path"] == "sda1"


def test_find_partition_in_dict_payload() -> None:
    """Partitions serialized as a keyed map (live backend shape) are found."""
    disks = [{"id": "d1", "partitions": {"p1": {"id": "p1", "device_path": "sda1"}}}]
    found = VolumeRepository.find_partition(disks, "p1")
    assert found is not None
    assert found["device_path"] == "sda1"


def test_find_partition_uses_map_key_when_id_omitted() -> None:
    """A map value without ``id`` falls back to its key (the partition id)."""
    disks = [{"id": "d1", "partitions": {"p1": {"device_path": "sda1"}}}]
    found = VolumeRepository.find_partition(disks, "p1")
    assert found is not None
    assert found["id"] == "p1"
    assert found["device_path"] == "sda1"


def test_find_partition_tolerates_invalid_containers() -> None:
    """None/other invalid partitions containers never raise."""
    assert VolumeRepository.find_partition(None, "p1") is None
    assert VolumeRepository.find_partition("nope", "p1") is None
    assert (
        VolumeRepository.find_partition([{"id": "d1", "partitions": None}], "p1")
        is None
    )
    assert (
        VolumeRepository.find_partition([{"id": "d1", "partitions": 42}], "p1") is None
    )
    assert (
        VolumeRepository.find_partition(
            [{"id": "d1", "partitions": {"p1": None}}], "p1"
        )
        is None
    )


def test_find_disk_io_tolerates_invalid_container() -> None:
    """None disk_io container or non-dict entries return None instead of raising."""
    assert VolumeRepository.find_disk_io(None, "sda") is None
    assert VolumeRepository.find_disk_io({"disk_io": None}, "sda") is None
    assert VolumeRepository.find_disk_io({"disk_io": "x"}, "sda") is None
    assert VolumeRepository.find_disk_io({"disk_io": {"sda": None}}, "sda") is None
    stats = VolumeRepository.find_disk_io(
        {"disk_io": {"sda": {"read_bytes": 1}}}, "sda"
    )
    assert stats == {"read_bytes": 1}


def test_find_partition_health_tolerates_invalid_container() -> None:
    """None partition_health container or non-dict entries return None."""
    assert VolumeRepository.find_partition_health(None, "/dev/sda1") is None
    assert (
        VolumeRepository.find_partition_health({"partition_health": None}, "d") is None
    )
    assert (
        VolumeRepository.find_partition_health({"partition_health": "x"}, "d") is None
    )
    assert (
        VolumeRepository.find_partition_health({"partition_health": {"d": []}}, "d")
        is None
    )
    info = VolumeRepository.find_partition_health(
        {"partition_health": {"d": {"fstype": "ext4"}}}, "d"
    )
    assert info == {"fstype": "ext4"}


def test_find_disk_miss_and_invalid_input() -> None:
    """Unknown disk ids and non-list payloads return None."""
    disks = [{"id": "d1", "device": "sda"}]
    assert VolumeRepository.find_disk(disks, "d1") == {"id": "d1", "device": "sda"}
    assert VolumeRepository.find_disk(disks, "nope") is None
    assert VolumeRepository.find_disk(None, "d1") is None
    assert VolumeRepository.find_disk("nope", "d1") is None


def test_iter_partitions_non_dict_disk() -> None:
    """A non-dict disk yields no partitions instead of raising."""
    assert VolumeRepository.iter_partitions(None) == []
    assert VolumeRepository.iter_partitions("nope") == []
    assert VolumeRepository.iter_partitions([{"id": "d1"}]) == []


# -- VolumeRepository live-payload helpers (issue #1394) --


def test_display_device_prefers_device_path() -> None:
    """display_device resolves device_path first."""
    entry = {
        "device_path": "/dev/sda1",
        "legacy_device_path": "/dev/disk/by-id/ata-X-part1",
        "legacy_device_name": "sda1",
    }
    assert VolumeRepository.display_device(entry) == "/dev/sda1"


def test_display_device_falls_back_to_legacy_path() -> None:
    """display_device falls back to legacy_device_path when device_path missing."""
    entry = {
        "legacy_device_path": "/dev/disk/by-id/ata-X-part1",
        "legacy_device_name": "sda1",
    }
    assert VolumeRepository.display_device(entry) == "/dev/disk/by-id/ata-X-part1"


def test_display_device_falls_back_to_legacy_name() -> None:
    """display_device falls back to legacy_device_name when paths missing."""
    entry = {"legacy_device_name": "sda1"}
    assert VolumeRepository.display_device(entry) == "sda1"


def test_display_device_falls_back_to_legacy_device_key() -> None:
    """display_device accepts the legacy ``device`` key for backward compat."""
    entry = {"device": "/dev/sda"}
    assert VolumeRepository.display_device(entry) == "/dev/sda"


def test_display_device_falls_back_to_id() -> None:
    """display_device falls back to the supplied id when no device key exists."""
    entry = {"id": "disk-001"}
    assert VolumeRepository.display_device(entry, "disk-001") == "disk-001"
    assert VolumeRepository.display_device(None, "fallback") == "fallback"


def test_partition_mounts_from_mount_point_data() -> None:
    """partition_mounts extracts entries from the mount_point_data map."""
    partition = {
        "mount_point_data": {
            "/mnt/data": {"path": "/mnt/data", "is_mounted": True},
            "/mnt/backup": {"path": "/mnt/backup", "is_mounted": False},
        }
    }
    mounts = VolumeRepository.partition_mounts(partition)
    assert len(mounts) == 2


def test_partition_mounts_falls_back_to_legacy_list() -> None:
    """partition_mounts accepts the legacy ``mounts`` list."""
    partition = {"mounts": ["/mnt/data"]}
    assert VolumeRepository.partition_mounts(partition) == ["/mnt/data"]


def test_partition_mounts_empty_when_no_data() -> None:
    """partition_mounts returns empty list when no mount data exists."""
    assert VolumeRepository.partition_mounts({}) == []
    assert VolumeRepository.partition_mounts(None) == []


def test_partition_shares_from_mount_point_data() -> None:
    """partition_shares extracts enabled shares from mount_point_data entries."""
    partition = {
        "mount_point_data": {
            "/mnt/data": {
                "path": "/mnt/data",
                "share": {"name": "share1", "disabled": False},
            },
            "/mnt/backup": {
                "path": "/mnt/backup",
                "share": {"name": "share2", "disabled": True},
            },
        }
    }
    shares = VolumeRepository.partition_shares(partition)
    assert len(shares) == 1
    assert shares[0]["name"] == "share1"


def test_partition_shares_falls_back_to_legacy_list() -> None:
    """partition_shares accepts the legacy ``shares`` list."""
    partition = {"shares": ["share1"]}
    assert VolumeRepository.partition_shares(partition) == ["share1"]


def test_partition_shares_empty_when_no_data() -> None:
    """partition_shares returns empty list when no share data exists."""
    assert VolumeRepository.partition_shares({}) == []
    assert VolumeRepository.partition_shares(None) == []


def test_is_partition_mounted_true_when_any_mounted() -> None:
    """is_partition_mounted returns True when any mount entry is mounted."""
    partition = {
        "mount_point_data": {
            "/mnt/data": {"is_mounted": False},
            "/mnt/backup": {"is_mounted": True},
        }
    }
    assert VolumeRepository.is_partition_mounted(partition) is True


def test_is_partition_mounted_false_when_none_mounted() -> None:
    """is_partition_mounted returns False when no mount entry is mounted."""
    partition = {
        "mount_point_data": {
            "/mnt/data": {"is_mounted": False},
        }
    }
    assert VolumeRepository.is_partition_mounted(partition) is False


def test_is_partition_mounted_falls_back_to_legacy_list() -> None:
    """is_partition_mounted accepts the legacy ``mounts`` list."""
    assert VolumeRepository.is_partition_mounted({"mounts": ["/mnt/data"]}) is True
    assert VolumeRepository.is_partition_mounted({"mounts": []}) is False
    assert VolumeRepository.is_partition_mounted({}) is False
    assert VolumeRepository.is_partition_mounted(None) is False
