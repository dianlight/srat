"""Sensor platform for SRAT Companion."""
from __future__ import annotations

from typing import Any

from homeassistant.components.sensor import SensorEntity
from homeassistant.config_entries import ConfigEntry
from homeassistant.const import CONF_HOST, CONF_PORT
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddEntitiesCallback
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from .const import DOMAIN, MANUFACTURER, MODEL
from .coordinator import SratCoordinator


async def async_setup_entry(
    hass: HomeAssistant,
    entry: ConfigEntry,
    async_add_entities: AddEntitiesCallback,
) -> None:
    """Set up SRAT Companion sensor based on a config entry."""
    coordinator = hass.data[DOMAIN][entry.entry_id]
    
    entities = [
        SratConnectionSensor(coordinator, entry),
        SratEventCountSensor(coordinator, entry),
    ]
    
    async_add_entities(entities)


class SratBaseSensor(CoordinatorEntity, SensorEntity):
    """Base class for SRAT sensors."""

    def __init__(self, coordinator: SratCoordinator, entry: ConfigEntry) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator)
        self.entry = entry
        self.host = entry.data[CONF_HOST]
        self.port = entry.data[CONF_PORT]

    @property
    def device_info(self) -> dict[str, Any]:
        """Return device information."""
        return {
            "identifiers": {(DOMAIN, f"{self.host}:{self.port}")},
            "name": f"SRAT Companion {self.host}:{self.port}",
            "manufacturer": MANUFACTURER,
            "model": MODEL,
            "sw_version": "1.0.0",
        }


class SratConnectionSensor(SratBaseSensor):
    """Sensor for SRAT connection status."""

    def __init__(self, coordinator: SratCoordinator, entry: ConfigEntry) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._attr_name = "SRAT Connection"
        self._attr_unique_id = f"{self.host}_{self.port}_connection"

    @property
    def state(self) -> str:
        """Return the state of the sensor."""
        return "connected" if self.coordinator.connected else "disconnected"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return additional state attributes."""
        return {
            "host": self.host,
            "port": self.port,
            "last_event_time": self.coordinator.last_update_success_time,
        }


class SratEventCountSensor(SratBaseSensor):
    """Sensor for SRAT event count."""

    def __init__(self, coordinator: SratCoordinator, entry: ConfigEntry) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._attr_name = "SRAT Event Count"
        self._attr_unique_id = f"{self.host}_{self.port}_event_count"

    @property
    def state(self) -> int:
        """Return the state of the sensor."""
        if self.coordinator.data and "events" in self.coordinator.data:
            return len(self.coordinator.data["events"])
        return 0

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return additional state attributes."""
        attrs = {
            "host": self.host,
            "port": self.port,
        }
        
        if self.coordinator.data and "events" in self.coordinator.data:
            events = self.coordinator.data["events"]
            if events:
                attrs["last_event"] = events[-1]
        
        return attrs
