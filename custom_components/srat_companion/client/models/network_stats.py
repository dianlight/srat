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
    from ..models.global_nic_stats import GlobalNicStats
    from ..models.nic_io_stats import NicIOStats


T = TypeVar("T", bound="NetworkStats")


@_attrs_define
class NetworkStats:
    """

    Attributes:
        global_ (GlobalNicStats):
        per_nic_io (Union[None, list['NicIOStats']]):

    """

    global_: "GlobalNicStats"
    per_nic_io: None | list["NicIOStats"]

    def to_dict(self) -> dict[str, Any]:
        global_ = self.global_.to_dict()

        per_nic_io: None | list[dict[str, Any]]
        if isinstance(self.per_nic_io, list):
            per_nic_io = []
            for per_nic_io_type_0_item_data in self.per_nic_io:
                per_nic_io_type_0_item = per_nic_io_type_0_item_data.to_dict()
                per_nic_io.append(per_nic_io_type_0_item)

        else:
            per_nic_io = self.per_nic_io

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "global": global_,
                "perNicIO": per_nic_io,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.global_nic_stats import GlobalNicStats
        from ..models.nic_io_stats import NicIOStats

        d = dict(src_dict)
        global_ = GlobalNicStats.from_dict(d.pop("global"))

        def _parse_per_nic_io(data: object) -> None | list["NicIOStats"]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                per_nic_io_type_0 = []
                _per_nic_io_type_0 = data
                for per_nic_io_type_0_item_data in _per_nic_io_type_0:
                    per_nic_io_type_0_item = NicIOStats.from_dict(
                        per_nic_io_type_0_item_data
                    )

                    per_nic_io_type_0.append(per_nic_io_type_0_item)

                return per_nic_io_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[NicIOStats]", data)

        per_nic_io = _parse_per_nic_io(d.pop("perNicIO"))

        network_stats = cls(
            global_=global_,
            per_nic_io=per_nic_io,
        )

        return network_stats
