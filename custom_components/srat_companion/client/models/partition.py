from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.mount_point_data import MountPointData


T = TypeVar("T", bound="Partition")


@_attrs_define
class Partition:
    """

    Attributes:
        device (Union[Unset, str]):
        host_mount_point_data (Union[Unset, list['MountPointData']]):
        id (Union[Unset, str]):
        mount_point_data (Union[Unset, list['MountPointData']]):
        name (Union[Unset, str]):
        size (Union[Unset, int]):
        system (Union[Unset, bool]):

    """

    device: Unset | str = UNSET
    host_mount_point_data: Unset | list["MountPointData"] = UNSET
    id: Unset | str = UNSET
    mount_point_data: Unset | list["MountPointData"] = UNSET
    name: Unset | str = UNSET
    size: Unset | int = UNSET
    system: Unset | bool = UNSET

    def to_dict(self) -> dict[str, Any]:
        device = self.device

        host_mount_point_data: Unset | list[dict[str, Any]] = UNSET
        if not isinstance(self.host_mount_point_data, Unset):
            host_mount_point_data = []
            for host_mount_point_data_item_data in self.host_mount_point_data:
                host_mount_point_data_item = host_mount_point_data_item_data.to_dict()
                host_mount_point_data.append(host_mount_point_data_item)

        id = self.id

        mount_point_data: Unset | list[dict[str, Any]] = UNSET
        if not isinstance(self.mount_point_data, Unset):
            mount_point_data = []
            for mount_point_data_item_data in self.mount_point_data:
                mount_point_data_item = mount_point_data_item_data.to_dict()
                mount_point_data.append(mount_point_data_item)

        name = self.name

        size = self.size

        system = self.system

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if device is not UNSET:
            field_dict["device"] = device
        if host_mount_point_data is not UNSET:
            field_dict["host_mount_point_data"] = host_mount_point_data
        if id is not UNSET:
            field_dict["id"] = id
        if mount_point_data is not UNSET:
            field_dict["mount_point_data"] = mount_point_data
        if name is not UNSET:
            field_dict["name"] = name
        if size is not UNSET:
            field_dict["size"] = size
        if system is not UNSET:
            field_dict["system"] = system

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.mount_point_data import MountPointData

        d = dict(src_dict)
        device = d.pop("device", UNSET)

        host_mount_point_data = []
        _host_mount_point_data = d.pop("host_mount_point_data", UNSET)
        for host_mount_point_data_item_data in _host_mount_point_data or []:
            host_mount_point_data_item = MountPointData.from_dict(
                host_mount_point_data_item_data
            )

            host_mount_point_data.append(host_mount_point_data_item)

        id = d.pop("id", UNSET)

        mount_point_data = []
        _mount_point_data = d.pop("mount_point_data", UNSET)
        for mount_point_data_item_data in _mount_point_data or []:
            mount_point_data_item = MountPointData.from_dict(mount_point_data_item_data)

            mount_point_data.append(mount_point_data_item)

        name = d.pop("name", UNSET)

        size = d.pop("size", UNSET)

        system = d.pop("system", UNSET)

        partition = cls(
            device=device,
            host_mount_point_data=host_mount_point_data,
            id=id,
            mount_point_data=mount_point_data,
            name=name,
            size=size,
            system=system,
        )

        return partition
