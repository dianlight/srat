from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
    Union,
    cast,
)

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.shared_resource_usage import SharedResourceUsage
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.mount_point_data import MountPointData
    from ..models.user import User


T = TypeVar("T", bound="SharedResource")


@_attrs_define
class SharedResource:
    """

    Attributes:
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.
        disabled (Union[Unset, bool]):
        ha_status (Union[Unset, str]):
        invalid (Union[Unset, bool]):
        is_ha_mounted (Union[Unset, bool]):
        mount_point_data (Union[Unset, MountPointData]):
        name (Union[Unset, str]):
        recycle_bin_enabled (Union[Unset, bool]):
        ro_users (Union[None, Unset, list['User']]):
        timemachine (Union[Unset, bool]):
        usage (Union[Unset, SharedResourceUsage]):
        users (Union[None, Unset, list['User']]):
        veto_files (Union[Unset, list[str]]):

    """

    schema: Unset | str = UNSET
    disabled: Unset | bool = UNSET
    ha_status: Unset | str = UNSET
    invalid: Unset | bool = UNSET
    is_ha_mounted: Unset | bool = UNSET
    mount_point_data: Union[Unset, "MountPointData"] = UNSET
    name: Unset | str = UNSET
    recycle_bin_enabled: Unset | bool = UNSET
    ro_users: None | Unset | list["User"] = UNSET
    timemachine: Unset | bool = UNSET
    usage: Unset | SharedResourceUsage = UNSET
    users: None | Unset | list["User"] = UNSET
    veto_files: Unset | list[str] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        disabled = self.disabled

        ha_status = self.ha_status

        invalid = self.invalid

        is_ha_mounted = self.is_ha_mounted

        mount_point_data: Unset | dict[str, Any] = UNSET
        if not isinstance(self.mount_point_data, Unset):
            mount_point_data = self.mount_point_data.to_dict()

        name = self.name

        recycle_bin_enabled = self.recycle_bin_enabled

        ro_users: None | Unset | list[dict[str, Any]]
        if isinstance(self.ro_users, Unset):
            ro_users = UNSET
        elif isinstance(self.ro_users, list):
            ro_users = []
            for ro_users_type_0_item_data in self.ro_users:
                ro_users_type_0_item = ro_users_type_0_item_data.to_dict()
                ro_users.append(ro_users_type_0_item)

        else:
            ro_users = self.ro_users

        timemachine = self.timemachine

        usage: Unset | str = UNSET
        if not isinstance(self.usage, Unset):
            usage = self.usage.value

        users: None | Unset | list[dict[str, Any]]
        if isinstance(self.users, Unset):
            users = UNSET
        elif isinstance(self.users, list):
            users = []
            for users_type_0_item_data in self.users:
                users_type_0_item = users_type_0_item_data.to_dict()
                users.append(users_type_0_item)

        else:
            users = self.users

        veto_files: Unset | list[str] = UNSET
        if not isinstance(self.veto_files, Unset):
            veto_files = self.veto_files

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if disabled is not UNSET:
            field_dict["disabled"] = disabled
        if ha_status is not UNSET:
            field_dict["ha_status"] = ha_status
        if invalid is not UNSET:
            field_dict["invalid"] = invalid
        if is_ha_mounted is not UNSET:
            field_dict["is_ha_mounted"] = is_ha_mounted
        if mount_point_data is not UNSET:
            field_dict["mount_point_data"] = mount_point_data
        if name is not UNSET:
            field_dict["name"] = name
        if recycle_bin_enabled is not UNSET:
            field_dict["recycle_bin_enabled"] = recycle_bin_enabled
        if ro_users is not UNSET:
            field_dict["ro_users"] = ro_users
        if timemachine is not UNSET:
            field_dict["timemachine"] = timemachine
        if usage is not UNSET:
            field_dict["usage"] = usage
        if users is not UNSET:
            field_dict["users"] = users
        if veto_files is not UNSET:
            field_dict["veto_files"] = veto_files

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.mount_point_data import MountPointData
        from ..models.user import User

        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        disabled = d.pop("disabled", UNSET)

        ha_status = d.pop("ha_status", UNSET)

        invalid = d.pop("invalid", UNSET)

        is_ha_mounted = d.pop("is_ha_mounted", UNSET)

        _mount_point_data = d.pop("mount_point_data", UNSET)
        mount_point_data: Unset | MountPointData
        if isinstance(_mount_point_data, Unset):
            mount_point_data = UNSET
        else:
            mount_point_data = MountPointData.from_dict(_mount_point_data)

        name = d.pop("name", UNSET)

        recycle_bin_enabled = d.pop("recycle_bin_enabled", UNSET)

        def _parse_ro_users(data: object) -> None | Unset | list["User"]:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                ro_users_type_0 = []
                _ro_users_type_0 = data
                for ro_users_type_0_item_data in _ro_users_type_0:
                    ro_users_type_0_item = User.from_dict(ro_users_type_0_item_data)

                    ro_users_type_0.append(ro_users_type_0_item)

                return ro_users_type_0
            except:  # noqa: E722
                pass
            return cast("None | Unset | list[User]", data)

        ro_users = _parse_ro_users(d.pop("ro_users", UNSET))

        timemachine = d.pop("timemachine", UNSET)

        _usage = d.pop("usage", UNSET)
        usage: Unset | SharedResourceUsage
        if isinstance(_usage, Unset):
            usage = UNSET
        else:
            usage = SharedResourceUsage(_usage)

        def _parse_users(data: object) -> None | Unset | list["User"]:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                users_type_0 = []
                _users_type_0 = data
                for users_type_0_item_data in _users_type_0:
                    users_type_0_item = User.from_dict(users_type_0_item_data)

                    users_type_0.append(users_type_0_item)

                return users_type_0
            except:  # noqa: E722
                pass
            return cast("None | Unset | list[User]", data)

        users = _parse_users(d.pop("users", UNSET))

        veto_files = cast("list[str]", d.pop("veto_files", UNSET))

        shared_resource = cls(
            schema=schema,
            disabled=disabled,
            ha_status=ha_status,
            invalid=invalid,
            is_ha_mounted=is_ha_mounted,
            mount_point_data=mount_point_data,
            name=name,
            recycle_bin_enabled=recycle_bin_enabled,
            ro_users=ro_users,
            timemachine=timemachine,
            usage=usage,
            users=users,
            veto_files=veto_files,
        )

        shared_resource.additional_properties = d
        return shared_resource

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
