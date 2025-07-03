from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="GlobalDiskStats")


@_attrs_define
class GlobalDiskStats:
    """

    Attributes:
        total_iops (float):
        total_read_latency_ms (float):
        total_write_latency_ms (float):

    """

    total_iops: float
    total_read_latency_ms: float
    total_write_latency_ms: float

    def to_dict(self) -> dict[str, Any]:
        total_iops = self.total_iops

        total_read_latency_ms = self.total_read_latency_ms

        total_write_latency_ms = self.total_write_latency_ms

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "total_iops": total_iops,
                "total_read_latency_ms": total_read_latency_ms,
                "total_write_latency_ms": total_write_latency_ms,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        total_iops = d.pop("total_iops")

        total_read_latency_ms = d.pop("total_read_latency_ms")

        total_write_latency_ms = d.pop("total_write_latency_ms")

        global_disk_stats = cls(
            total_iops=total_iops,
            total_read_latency_ms=total_read_latency_ms,
            total_write_latency_ms=total_write_latency_ms,
        )

        return global_disk_stats
