"""Sensor platform for the SRAT integration."""

from __future__ import annotations

import logging
import re
from typing import Any

from homeassistant.components.sensor import (
    SensorDeviceClass,
    SensorEntity,
    SensorStateClass,
)
from homeassistant.const import UnitOfInformation
from homeassistant.core import HomeAssistant
from homeassistant.helpers.device_registry import DeviceInfo
from homeassistant.helpers.entity_platform import AddEntitiesCallback
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from . import SRATConfigEntry
from .const import DOMAIN
from .coordinator import SRATDataCoordinator

_LOGGER = logging.getLogger(__name__)

_NON_ALNUM = re.compile(r"[^a-zA-Z0-9]+")


def _sanitize_id(value: str) -> str:
    """Sanitize a string to make it suitable for use in entity IDs."""
    return _NON_ALNUM.sub("_", value).lower().strip("_")


async def async_setup_entry(
    hass: HomeAssistant,
    entry: SRATConfigEntry,
    async_add_entities: AddEntitiesCallback,
) -> None:
    """Set up SRAT sensors from a config entry."""
    coordinator = entry.runtime_data.coordinator

    entities: list[SensorEntity] = [
        SRATSambaStatusSensor(coordinator, entry),
        SRATSambaProcessStatusSensor(coordinator, entry),
        SRATVolumeStatusSensor(coordinator, entry),
        SRATGlobalDiskHealthSensor(coordinator, entry),
    ]

    # Add dynamic disk and partition sensors based on initial data
    if coordinator.data:
        disks = coordinator.data.get("disks", [])
        if isinstance(disks, list):
            for disk in disks:
                if isinstance(disk, dict):
                    entities.append(SRATDiskSensor(coordinator, entry, disk))
                    for partition in disk.get("partitions", []):
                        if isinstance(partition, dict):
                            entities.append(
                                SRATPartitionSensor(coordinator, entry, partition, disk)
                            )

        health = coordinator.data.get("disk_health")
        if isinstance(health, dict):
            for device_name, stats in health.get("disk_io", {}).items():
                if isinstance(stats, dict):
                    entities.append(
                        SRATDiskIOSensor(coordinator, entry, device_name, stats)
                    )
            for device, info in health.get("partition_health", {}).items():
                if isinstance(info, dict):
                    entities.append(
                        SRATPartitionHealthSensor(coordinator, entry, device, info)
                    )

    async_add_entities(entities, update_before_add=False)


class SRATSensorBase(CoordinatorEntity[SRATDataCoordinator], SensorEntity):
    """Base class for SRAT sensors."""

    _attr_has_entity_name = True

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator)
        self._entry = entry
        self._attr_device_info = DeviceInfo(
            identifiers={(DOMAIN, entry.entry_id)},
            name="SRAT",
            manufacturer="SRAT",
            model="SambaNAS REST Administration Tool",
            configuration_url=f"http://{entry.data.get('host', 'localhost')}:{entry.data.get('port', 8099)}",
        )


class SRATSambaStatusSensor(SRATSensorBase):
    """Sensor for Samba connection status."""

    _attr_name = "Samba Status"
    _attr_icon = "mdi:folder-network"

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._attr_unique_id = f"{entry.entry_id}_samba_status"

    @property
    def native_value(self) -> str | None:
        """Return the samba connection state."""
        status = (
            self.coordinator.data.get("samba_status") if self.coordinator.data else None
        )
        if not isinstance(status, dict):
            return None
        sessions = status.get("sessions", [])
        return "connected" if sessions else "idle"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return additional attributes."""
        status = (
            self.coordinator.data.get("samba_status") if self.coordinator.data else None
        )
        if not isinstance(status, dict):
            return {}
        return {
            "version": status.get("version"),
            "session_count": len(status.get("sessions", [])),
            "tcon_count": len(status.get("tcons", [])),
        }


class SRATSambaProcessStatusSensor(SRATSensorBase):
    """Sensor for Samba process status."""

    _attr_name = "Samba Process Status"
    _attr_icon = "mdi:cog"

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._attr_unique_id = f"{entry.entry_id}_samba_process_status"

    @property
    def native_value(self) -> str | None:
        """Return the overall process state."""
        status = (
            self.coordinator.data.get("process_status")
            if self.coordinator.data
            else None
        )
        if not isinstance(status, dict):
            return None
        running_count = sum(
            1
            for proc in status.values()
            if isinstance(proc, dict) and proc.get("is_running")
        )
        total = len(status)
        if running_count >= total:
            return "running"
        if running_count > 0:
            return "partial"
        return "stopped"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return per-process attributes."""
        status = (
            self.coordinator.data.get("process_status")
            if self.coordinator.data
            else None
        )
        if not isinstance(status, dict):
            return {}
        attrs: dict[str, Any] = {}
        for name, proc in status.items():
            if isinstance(proc, dict):
                attrs[f"{name}_running"] = proc.get("is_running", False)
                if proc.get("is_running"):
                    attrs[f"{name}_pid"] = proc.get("pid")
                    attrs[f"{name}_cpu_percent"] = proc.get("cpu_percent")
                    attrs[f"{name}_memory_percent"] = proc.get("memory_percent")
        return attrs


class SRATVolumeStatusSensor(SRATSensorBase):
    """Sensor for overall volume status."""

    _attr_name = "Volume Status"
    _attr_icon = "mdi:harddisk"

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._attr_unique_id = f"{entry.entry_id}_volume_status"

    @property
    def native_value(self) -> int | None:
        """Return the total number of disks."""
        disks = self.coordinator.data.get("disks") if self.coordinator.data else None
        if not isinstance(disks, list):
            return None
        return len(disks)

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return disk and partition counts."""
        disks = self.coordinator.data.get("disks") if self.coordinator.data else None
        if not isinstance(disks, list):
            return {}
        partition_count = sum(
            len(d.get("partitions", [])) for d in disks if isinstance(d, dict)
        )
        return {
            "disk_count": len(disks),
            "partition_count": partition_count,
        }


class SRATDiskSensor(SRATSensorBase):
    """Sensor for an individual disk."""

    _attr_icon = "mdi:harddisk"

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
        disk: dict[str, Any],
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        disk_id = _sanitize_id(disk.get("id", "unknown"))
        self._disk_id = disk.get("id", "unknown")
        self._attr_unique_id = f"{entry.entry_id}_disk_{disk_id}"
        self._attr_name = f"Disk {disk.get('device', disk_id)}"

    @property
    def native_value(self) -> str:
        """Return disk connection state."""
        return "connected"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return disk attributes."""
        disk = self._find_disk()
        if not isinstance(disk, dict):
            return {}
        size_bytes = disk.get("size", 0)
        return {
            "device": disk.get("device"),
            "model": disk.get("model"),
            "vendor": disk.get("vendor"),
            "serial": disk.get("serial"),
            "size_bytes": size_bytes,
            "size_gb": round(size_bytes / (1024**3), 2) if size_bytes else 0,
            "connection_bus": disk.get("connectionBus"),
            "removable": disk.get("removable"),
            "partition_count": len(disk.get("partitions", [])),
        }

    def _find_disk(self) -> dict[str, Any] | None:
        """Find the disk in current coordinator data."""
        disks = self.coordinator.data.get("disks") if self.coordinator.data else None
        if not isinstance(disks, list):
            return None
        for disk in disks:
            if isinstance(disk, dict) and disk.get("id") == self._disk_id:
                return disk
        return None


class SRATPartitionSensor(SRATSensorBase):
    """Sensor for an individual partition."""

    _attr_icon = "mdi:folder"

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
        partition: dict[str, Any],
        disk: dict[str, Any],
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        part_id = _sanitize_id(partition.get("id", "unknown"))
        self._partition_id = partition.get("id", "unknown")
        self._disk_id = disk.get("id", "unknown")
        self._attr_unique_id = f"{entry.entry_id}_partition_{part_id}"
        self._attr_name = f"Partition {partition.get('device', part_id)}"

    @property
    def native_value(self) -> str | None:
        """Return partition state."""
        partition = self._find_partition()
        if not isinstance(partition, dict):
            return None
        shares = partition.get("shares", [])
        mounts = partition.get("mounts", [])
        if shares:
            return "shared"
        if mounts:
            return "mounted"
        return "unmounted"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return partition attributes."""
        partition = self._find_partition()
        if not isinstance(partition, dict):
            return {}
        size_bytes = partition.get("size", 0)
        return {
            "disk_id": self._disk_id,
            "device": partition.get("device"),
            "name": partition.get("name"),
            "size_bytes": size_bytes,
            "size_gb": round(size_bytes / (1024**3), 2) if size_bytes else 0,
            "system": partition.get("system"),
            "mount_count": len(partition.get("mounts", [])),
            "share_count": len(partition.get("shares", [])),
        }

    def _find_partition(self) -> dict[str, Any] | None:
        """Find the partition in current coordinator data."""
        disks = self.coordinator.data.get("disks") if self.coordinator.data else None
        if not isinstance(disks, list):
            return None
        for disk in disks:
            if not isinstance(disk, dict):
                continue
            for part in disk.get("partitions", []):
                if isinstance(part, dict) and part.get("id") == self._partition_id:
                    return part
        return None


class SRATGlobalDiskHealthSensor(SRATSensorBase):
    """Sensor for global disk health metrics."""

    _attr_name = "Global Disk Health"
    _attr_icon = "mdi:heart-pulse"
    _attr_state_class = SensorStateClass.MEASUREMENT

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._attr_unique_id = f"{entry.entry_id}_global_disk_health"

    @property
    def native_value(self) -> float | None:
        """Return total IOPS."""
        health = (
            self.coordinator.data.get("disk_health") if self.coordinator.data else None
        )
        if not isinstance(health, dict):
            return None
        global_stats = health.get("global", {})
        if not isinstance(global_stats, dict):
            return None
        return global_stats.get("total_iops")

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return global disk health attributes."""
        health = (
            self.coordinator.data.get("disk_health") if self.coordinator.data else None
        )
        if not isinstance(health, dict):
            return {}
        global_stats = health.get("global", {})
        if not isinstance(global_stats, dict):
            return {}
        return {
            "total_read_iops": global_stats.get("total_read_iops"),
            "total_write_iops": global_stats.get("total_write_iops"),
            "avg_read_latency_ms": global_stats.get("avg_read_latency_ms"),
            "avg_write_latency_ms": global_stats.get("avg_write_latency_ms"),
        }


class SRATDiskIOSensor(SRATSensorBase):
    """Sensor for per-disk I/O statistics."""

    _attr_icon = "mdi:chart-line"
    _attr_state_class = SensorStateClass.MEASUREMENT

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
        device_name: str,
        stats: dict[str, Any],
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        safe_name = _sanitize_id(device_name)
        self._device_name = device_name
        self._attr_unique_id = f"{entry.entry_id}_disk_io_{safe_name}"
        self._attr_name = f"Disk IO {device_name}"

    @property
    def native_value(self) -> float | None:
        """Return total IOPS for this disk."""
        stats = self._find_stats()
        if not isinstance(stats, dict):
            return None
        read_iops = stats.get("read_iops", 0) or 0
        write_iops = stats.get("write_iops", 0) or 0
        return read_iops + write_iops

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return disk I/O attributes."""
        stats = self._find_stats()
        if not isinstance(stats, dict):
            return {}
        return {
            "device_name": self._device_name,
            "read_iops": stats.get("read_iops"),
            "write_iops": stats.get("write_iops"),
            "read_latency_ms": stats.get("read_latency_ms"),
            "write_latency_ms": stats.get("write_latency_ms"),
            "smart_temperature": stats.get("smart_temperature"),
            "smart_power_on_hours": stats.get("smart_power_on_hours"),
            "smart_power_cycle_count": stats.get("smart_power_cycle_count"),
        }

    def _find_stats(self) -> dict[str, Any] | None:
        """Find disk IO stats in current data."""
        health = (
            self.coordinator.data.get("disk_health") if self.coordinator.data else None
        )
        if not isinstance(health, dict):
            return None
        return health.get("disk_io", {}).get(self._device_name)


class SRATPartitionHealthSensor(SRATSensorBase):
    """Sensor for per-partition health information."""

    _attr_icon = "mdi:database"
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_native_unit_of_measurement = UnitOfInformation.BYTES
    _attr_device_class = SensorDeviceClass.DATA_SIZE

    def __init__(
        self,
        coordinator: SRATDataCoordinator,
        entry: SRATConfigEntry,
        device: str,
        info: dict[str, Any],
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        safe_device = _sanitize_id(device)
        self._device = device
        self._attr_unique_id = f"{entry.entry_id}_partition_health_{safe_device}"
        self._attr_name = f"Partition Health {device}"

    @property
    def native_value(self) -> int | None:
        """Return free space in bytes."""
        info = self._find_info()
        if not isinstance(info, dict):
            return None
        return info.get("free_space_bytes")

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return partition health attributes."""
        info = self._find_info()
        if not isinstance(info, dict):
            return {}
        return {
            "device": self._device,
            "mount_point": info.get("mount_point"),
            "fstype": info.get("fstype"),
            "total_space_bytes": info.get("total_space_bytes"),
            "free_space_bytes": info.get("free_space_bytes"),
            "usage_percent": info.get("usage_percent"),
            "fsck_needed": info.get("fsck_needed"),
            "fsck_supported": info.get("fsck_supported"),
            "disk_name": info.get("disk_name"),
        }

    def _find_info(self) -> dict[str, Any] | None:
        """Find partition health info in current data."""
        health = (
            self.coordinator.data.get("disk_health") if self.coordinator.data else None
        )
        if not isinstance(health, dict):
            return None
        return health.get("partition_health", {}).get(self._device)
