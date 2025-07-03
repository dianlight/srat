from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="NicIOStats")


@_attrs_define
class NicIOStats:
    """

    Attributes:
        device_max_speed (int):
        device_name (str):
        inbound_traffic (float):
        outbound_traffic (float):

    """

    device_max_speed: int
    device_name: str
    inbound_traffic: float
    outbound_traffic: float

    def to_dict(self) -> dict[str, Any]:
        device_max_speed = self.device_max_speed

        device_name = self.device_name

        inbound_traffic = self.inbound_traffic

        outbound_traffic = self.outbound_traffic

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "deviceMaxSpeed": device_max_speed,
                "deviceName": device_name,
                "inboundTraffic": inbound_traffic,
                "outboundTraffic": outbound_traffic,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        device_max_speed = d.pop("deviceMaxSpeed")

        device_name = d.pop("deviceName")

        inbound_traffic = d.pop("inboundTraffic")

        outbound_traffic = d.pop("outboundTraffic")

        nic_io_stats = cls(
            device_max_speed=device_max_speed,
            device_name=device_name,
            inbound_traffic=inbound_traffic,
            outbound_traffic=outbound_traffic,
        )

        return nic_io_stats
