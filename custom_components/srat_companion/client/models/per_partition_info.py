from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="PerPartitionInfo")


@_attrs_define
class PerPartitionInfo:
    """

    Attributes:
        device (str):
        free_space_bytes (int):
        fsck_needed (bool):
        fsck_supported (bool):
        fstype (str):
        mount_point (str):
        total_space_bytes (int):

    """

    device: str
    free_space_bytes: int
    fsck_needed: bool
    fsck_supported: bool
    fstype: str
    mount_point: str
    total_space_bytes: int

    def to_dict(self) -> dict[str, Any]:
        device = self.device

        free_space_bytes = self.free_space_bytes

        fsck_needed = self.fsck_needed

        fsck_supported = self.fsck_supported

        fstype = self.fstype

        mount_point = self.mount_point

        total_space_bytes = self.total_space_bytes

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "device": device,
                "free_space_bytes": free_space_bytes,
                "fsck_needed": fsck_needed,
                "fsck_supported": fsck_supported,
                "fstype": fstype,
                "mount_point": mount_point,
                "total_space_bytes": total_space_bytes,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        device = d.pop("device")

        free_space_bytes = d.pop("free_space_bytes")

        fsck_needed = d.pop("fsck_needed")

        fsck_supported = d.pop("fsck_supported")

        fstype = d.pop("fstype")

        mount_point = d.pop("mount_point")

        total_space_bytes = d.pop("total_space_bytes")

        per_partition_info = cls(
            device=device,
            free_space_bytes=free_space_bytes,
            fsck_needed=fsck_needed,
            fsck_supported=fsck_supported,
            fstype=fstype,
            mount_point=mount_point,
            total_space_bytes=total_space_bytes,
        )

        return per_partition_info
