from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
    Union,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.smart_info import SmartInfo


T = TypeVar("T", bound="DiskIOStats")


@_attrs_define
class DiskIOStats:
    """

    Attributes:
        device_description (str):
        device_name (str):
        read_iops (float):
        read_latency_ms (float):
        write_iops (float):
        write_latency_ms (float):
        smart_data (Union[Unset, SmartInfo]):

    """

    device_description: str
    device_name: str
    read_iops: float
    read_latency_ms: float
    write_iops: float
    write_latency_ms: float
    smart_data: Union[Unset, "SmartInfo"] = UNSET

    def to_dict(self) -> dict[str, Any]:
        device_description = self.device_description

        device_name = self.device_name

        read_iops = self.read_iops

        read_latency_ms = self.read_latency_ms

        write_iops = self.write_iops

        write_latency_ms = self.write_latency_ms

        smart_data: Unset | dict[str, Any] = UNSET
        if not isinstance(self.smart_data, Unset):
            smart_data = self.smart_data.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "device_description": device_description,
                "device_name": device_name,
                "read_iops": read_iops,
                "read_latency_ms": read_latency_ms,
                "write_iops": write_iops,
                "write_latency_ms": write_latency_ms,
            }
        )
        if smart_data is not UNSET:
            field_dict["smart_data"] = smart_data

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.smart_info import SmartInfo

        d = dict(src_dict)
        device_description = d.pop("device_description")

        device_name = d.pop("device_name")

        read_iops = d.pop("read_iops")

        read_latency_ms = d.pop("read_latency_ms")

        write_iops = d.pop("write_iops")

        write_latency_ms = d.pop("write_latency_ms")

        _smart_data = d.pop("smart_data", UNSET)
        smart_data: Unset | SmartInfo
        if isinstance(_smart_data, Unset):
            smart_data = UNSET
        else:
            smart_data = SmartInfo.from_dict(_smart_data)

        disk_io_stats = cls(
            device_description=device_description,
            device_name=device_name,
            read_iops=read_iops,
            read_latency_ms=read_latency_ms,
            write_iops=write_iops,
            write_latency_ms=write_latency_ms,
            smart_data=smart_data,
        )

        return disk_io_stats
