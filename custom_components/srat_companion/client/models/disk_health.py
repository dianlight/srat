from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define

if TYPE_CHECKING:
    from ..models.disk_health_per_partition_info import DiskHealthPerPartitionInfo
    from ..models.disk_io_stats import DiskIOStats
    from ..models.global_disk_stats import GlobalDiskStats


T = TypeVar("T", bound="DiskHealth")


@_attrs_define
class DiskHealth:
    """

    Attributes:
        global_ (GlobalDiskStats):
        per_disk_io (Union[None, list['DiskIOStats']]):
        per_partition_info (DiskHealthPerPartitionInfo):

    """

    global_: "GlobalDiskStats"
    per_disk_io: None | list["DiskIOStats"]
    per_partition_info: "DiskHealthPerPartitionInfo"

    def to_dict(self) -> dict[str, Any]:
        global_ = self.global_.to_dict()

        per_disk_io: None | list[dict[str, Any]]
        if isinstance(self.per_disk_io, list):
            per_disk_io = []
            for per_disk_io_type_0_item_data in self.per_disk_io:
                per_disk_io_type_0_item = per_disk_io_type_0_item_data.to_dict()
                per_disk_io.append(per_disk_io_type_0_item)

        else:
            per_disk_io = self.per_disk_io

        per_partition_info = self.per_partition_info.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "global": global_,
                "per_disk_io": per_disk_io,
                "per_partition_info": per_partition_info,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.disk_health_per_partition_info import DiskHealthPerPartitionInfo
        from ..models.disk_io_stats import DiskIOStats
        from ..models.global_disk_stats import GlobalDiskStats

        d = dict(src_dict)
        global_ = GlobalDiskStats.from_dict(d.pop("global"))

        def _parse_per_disk_io(data: object) -> None | list["DiskIOStats"]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                per_disk_io_type_0 = []
                _per_disk_io_type_0 = data
                for per_disk_io_type_0_item_data in _per_disk_io_type_0:
                    per_disk_io_type_0_item = DiskIOStats.from_dict(
                        per_disk_io_type_0_item_data
                    )

                    per_disk_io_type_0.append(per_disk_io_type_0_item)

                return per_disk_io_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[DiskIOStats]", data)

        per_disk_io = _parse_per_disk_io(d.pop("per_disk_io"))

        per_partition_info = DiskHealthPerPartitionInfo.from_dict(
            d.pop("per_partition_info")
        )

        disk_health = cls(
            global_=global_,
            per_disk_io=per_disk_io,
            per_partition_info=per_partition_info,
        )

        return disk_health
