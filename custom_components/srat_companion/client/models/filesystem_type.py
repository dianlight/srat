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
    from ..models.mount_flag import MountFlag


T = TypeVar("T", bound="FilesystemType")


@_attrs_define
class FilesystemType:
    """

    Attributes:
        custom_mount_flags (Union[None, list['MountFlag']]):
        mount_flags (Union[None, list['MountFlag']]):
        name (str):
        type_ (str):

    """

    custom_mount_flags: None | list["MountFlag"]
    mount_flags: None | list["MountFlag"]
    name: str
    type_: str

    def to_dict(self) -> dict[str, Any]:
        custom_mount_flags: None | list[dict[str, Any]]
        if isinstance(self.custom_mount_flags, list):
            custom_mount_flags = []
            for custom_mount_flags_type_0_item_data in self.custom_mount_flags:
                custom_mount_flags_type_0_item = (
                    custom_mount_flags_type_0_item_data.to_dict()
                )
                custom_mount_flags.append(custom_mount_flags_type_0_item)

        else:
            custom_mount_flags = self.custom_mount_flags

        mount_flags: None | list[dict[str, Any]]
        if isinstance(self.mount_flags, list):
            mount_flags = []
            for mount_flags_type_0_item_data in self.mount_flags:
                mount_flags_type_0_item = mount_flags_type_0_item_data.to_dict()
                mount_flags.append(mount_flags_type_0_item)

        else:
            mount_flags = self.mount_flags

        name = self.name

        type_ = self.type_

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "customMountFlags": custom_mount_flags,
                "mountFlags": mount_flags,
                "name": name,
                "type": type_,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.mount_flag import MountFlag

        d = dict(src_dict)

        def _parse_custom_mount_flags(data: object) -> None | list["MountFlag"]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                custom_mount_flags_type_0 = []
                _custom_mount_flags_type_0 = data
                for custom_mount_flags_type_0_item_data in _custom_mount_flags_type_0:
                    custom_mount_flags_type_0_item = MountFlag.from_dict(
                        custom_mount_flags_type_0_item_data
                    )

                    custom_mount_flags_type_0.append(custom_mount_flags_type_0_item)

                return custom_mount_flags_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[MountFlag]", data)

        custom_mount_flags = _parse_custom_mount_flags(d.pop("customMountFlags"))

        def _parse_mount_flags(data: object) -> None | list["MountFlag"]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                mount_flags_type_0 = []
                _mount_flags_type_0 = data
                for mount_flags_type_0_item_data in _mount_flags_type_0:
                    mount_flags_type_0_item = MountFlag.from_dict(
                        mount_flags_type_0_item_data
                    )

                    mount_flags_type_0.append(mount_flags_type_0_item)

                return mount_flags_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[MountFlag]", data)

        mount_flags = _parse_mount_flags(d.pop("mountFlags"))

        name = d.pop("name")

        type_ = d.pop("type")

        filesystem_type = cls(
            custom_mount_flags=custom_mount_flags,
            mount_flags=mount_flags,
            name=name,
            type_=type_,
        )

        return filesystem_type
