from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define

from ..models.mount_point_data_type import MountPointDataType
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.mount_flag import MountFlag
    from ..models.shared_resource import SharedResource


T = TypeVar("T", bound="MountPointData")


@_attrs_define
class MountPointData:
    """

    Attributes:
        path (str):
        type_ (MountPointDataType):
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.
        custom_flags (Union[Unset, list['MountFlag']]):
        device (Union[Unset, str]):
        flags (Union[Unset, list['MountFlag']]):
        fstype (Union[Unset, str]):
        invalid (Union[Unset, bool]):
        invalid_error (Union[Unset, str]):
        is_mounted (Union[Unset, bool]):
        is_to_mount_at_startup (Union[Unset, bool]):
        path_hash (Union[Unset, str]):
        shares (Union[None, Unset, list['SharedResource']]):
        warnings (Union[Unset, str]):

    """

    path: str
    type_: MountPointDataType
    schema: Unset | str = UNSET
    custom_flags: Unset | list["MountFlag"] = UNSET
    device: Unset | str = UNSET
    flags: Unset | list["MountFlag"] = UNSET
    fstype: Unset | str = UNSET
    invalid: Unset | bool = UNSET
    invalid_error: Unset | str = UNSET
    is_mounted: Unset | bool = UNSET
    is_to_mount_at_startup: Unset | bool = UNSET
    path_hash: Unset | str = UNSET
    shares: None | Unset | list["SharedResource"] = UNSET
    warnings: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        path = self.path

        type_ = self.type_.value

        schema = self.schema

        custom_flags: Unset | list[dict[str, Any]] = UNSET
        if not isinstance(self.custom_flags, Unset):
            custom_flags = []
            for custom_flags_item_data in self.custom_flags:
                custom_flags_item = custom_flags_item_data.to_dict()
                custom_flags.append(custom_flags_item)

        device = self.device

        flags: Unset | list[dict[str, Any]] = UNSET
        if not isinstance(self.flags, Unset):
            flags = []
            for flags_item_data in self.flags:
                flags_item = flags_item_data.to_dict()
                flags.append(flags_item)

        fstype = self.fstype

        invalid = self.invalid

        invalid_error = self.invalid_error

        is_mounted = self.is_mounted

        is_to_mount_at_startup = self.is_to_mount_at_startup

        path_hash = self.path_hash

        shares: None | Unset | list[dict[str, Any]]
        if isinstance(self.shares, Unset):
            shares = UNSET
        elif isinstance(self.shares, list):
            shares = []
            for shares_type_0_item_data in self.shares:
                shares_type_0_item = shares_type_0_item_data.to_dict()
                shares.append(shares_type_0_item)

        else:
            shares = self.shares

        warnings = self.warnings

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "path": path,
                "type": type_,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if custom_flags is not UNSET:
            field_dict["custom_flags"] = custom_flags
        if device is not UNSET:
            field_dict["device"] = device
        if flags is not UNSET:
            field_dict["flags"] = flags
        if fstype is not UNSET:
            field_dict["fstype"] = fstype
        if invalid is not UNSET:
            field_dict["invalid"] = invalid
        if invalid_error is not UNSET:
            field_dict["invalid_error"] = invalid_error
        if is_mounted is not UNSET:
            field_dict["is_mounted"] = is_mounted
        if is_to_mount_at_startup is not UNSET:
            field_dict["is_to_mount_at_startup"] = is_to_mount_at_startup
        if path_hash is not UNSET:
            field_dict["path_hash"] = path_hash
        if shares is not UNSET:
            field_dict["shares"] = shares
        if warnings is not UNSET:
            field_dict["warnings"] = warnings

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.mount_flag import MountFlag
        from ..models.shared_resource import SharedResource

        d = dict(src_dict)
        path = d.pop("path")

        type_ = MountPointDataType(d.pop("type"))

        schema = d.pop("$schema", UNSET)

        custom_flags = []
        _custom_flags = d.pop("custom_flags", UNSET)
        for custom_flags_item_data in _custom_flags or []:
            custom_flags_item = MountFlag.from_dict(custom_flags_item_data)

            custom_flags.append(custom_flags_item)

        device = d.pop("device", UNSET)

        flags = []
        _flags = d.pop("flags", UNSET)
        for flags_item_data in _flags or []:
            flags_item = MountFlag.from_dict(flags_item_data)

            flags.append(flags_item)

        fstype = d.pop("fstype", UNSET)

        invalid = d.pop("invalid", UNSET)

        invalid_error = d.pop("invalid_error", UNSET)

        is_mounted = d.pop("is_mounted", UNSET)

        is_to_mount_at_startup = d.pop("is_to_mount_at_startup", UNSET)

        path_hash = d.pop("path_hash", UNSET)

        def _parse_shares(data: object) -> None | Unset | list["SharedResource"]:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                shares_type_0 = []
                _shares_type_0 = data
                for shares_type_0_item_data in _shares_type_0:
                    shares_type_0_item = SharedResource.from_dict(
                        shares_type_0_item_data
                    )

                    shares_type_0.append(shares_type_0_item)

                return shares_type_0
            except:  # noqa: E722
                pass
            return cast("None | Unset | list[SharedResource]", data)

        shares = _parse_shares(d.pop("shares", UNSET))

        warnings = d.pop("warnings", UNSET)

        mount_point_data = cls(
            path=path,
            type_=type_,
            schema=schema,
            custom_flags=custom_flags,
            device=device,
            flags=flags,
            fstype=fstype,
            invalid=invalid,
            invalid_error=invalid_error,
            is_mounted=is_mounted,
            is_to_mount_at_startup=is_to_mount_at_startup,
            path_hash=path_hash,
            shares=shares,
            warnings=warnings,
        )

        return mount_point_data
