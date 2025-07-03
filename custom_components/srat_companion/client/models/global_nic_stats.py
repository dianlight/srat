from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="GlobalNicStats")


@_attrs_define
class GlobalNicStats:
    """

    Attributes:
        total_inbound_traffic (float):
        total_outbound_traffic (float):

    """

    total_inbound_traffic: float
    total_outbound_traffic: float

    def to_dict(self) -> dict[str, Any]:
        total_inbound_traffic = self.total_inbound_traffic

        total_outbound_traffic = self.total_outbound_traffic

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "totalInboundTraffic": total_inbound_traffic,
                "totalOutboundTraffic": total_outbound_traffic,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        total_inbound_traffic = d.pop("totalInboundTraffic")

        total_outbound_traffic = d.pop("totalOutboundTraffic")

        global_nic_stats = cls(
            total_inbound_traffic=total_inbound_traffic,
            total_outbound_traffic=total_outbound_traffic,
        )

        return global_nic_stats
