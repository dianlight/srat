from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="AddonStatsData")


@_attrs_define
class AddonStatsData:
    """

    Attributes:
        blk_read (Union[Unset, int]):
        blk_write (Union[Unset, int]):
        cpu_percent (Union[Unset, float]):
        memory_limit (Union[Unset, int]):
        memory_percent (Union[Unset, float]):
        memory_usage (Union[Unset, int]):
        network_rx (Union[Unset, int]):
        network_tx (Union[Unset, int]):

    """

    blk_read: Unset | int = UNSET
    blk_write: Unset | int = UNSET
    cpu_percent: Unset | float = UNSET
    memory_limit: Unset | int = UNSET
    memory_percent: Unset | float = UNSET
    memory_usage: Unset | int = UNSET
    network_rx: Unset | int = UNSET
    network_tx: Unset | int = UNSET

    def to_dict(self) -> dict[str, Any]:
        blk_read = self.blk_read

        blk_write = self.blk_write

        cpu_percent = self.cpu_percent

        memory_limit = self.memory_limit

        memory_percent = self.memory_percent

        memory_usage = self.memory_usage

        network_rx = self.network_rx

        network_tx = self.network_tx

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if blk_read is not UNSET:
            field_dict["blk_read"] = blk_read
        if blk_write is not UNSET:
            field_dict["blk_write"] = blk_write
        if cpu_percent is not UNSET:
            field_dict["cpu_percent"] = cpu_percent
        if memory_limit is not UNSET:
            field_dict["memory_limit"] = memory_limit
        if memory_percent is not UNSET:
            field_dict["memory_percent"] = memory_percent
        if memory_usage is not UNSET:
            field_dict["memory_usage"] = memory_usage
        if network_rx is not UNSET:
            field_dict["network_rx"] = network_rx
        if network_tx is not UNSET:
            field_dict["network_tx"] = network_tx

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        blk_read = d.pop("blk_read", UNSET)

        blk_write = d.pop("blk_write", UNSET)

        cpu_percent = d.pop("cpu_percent", UNSET)

        memory_limit = d.pop("memory_limit", UNSET)

        memory_percent = d.pop("memory_percent", UNSET)

        memory_usage = d.pop("memory_usage", UNSET)

        network_rx = d.pop("network_rx", UNSET)

        network_tx = d.pop("network_tx", UNSET)

        addon_stats_data = cls(
            blk_read=blk_read,
            blk_write=blk_write,
            cpu_percent=cpu_percent,
            memory_limit=memory_limit,
            memory_percent=memory_percent,
            memory_usage=memory_usage,
            network_rx=network_rx,
            network_tx=network_tx,
        )

        return addon_stats_data
