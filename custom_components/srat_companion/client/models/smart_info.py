from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="SmartInfo")


@_attrs_define
class SmartInfo:
    """

    Attributes:
        power_cycle_count (int):
        power_on_hours (int):
        temperature (int):

    """

    power_cycle_count: int
    power_on_hours: int
    temperature: int

    def to_dict(self) -> dict[str, Any]:
        power_cycle_count = self.power_cycle_count

        power_on_hours = self.power_on_hours

        temperature = self.temperature

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "power_cycle_count": power_cycle_count,
                "power_on_hours": power_on_hours,
                "temperature": temperature,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        power_cycle_count = d.pop("power_cycle_count")

        power_on_hours = d.pop("power_on_hours")

        temperature = d.pop("temperature")

        smart_info = cls(
            power_cycle_count=power_cycle_count,
            power_on_hours=power_on_hours,
            temperature=temperature,
        )

        return smart_info
