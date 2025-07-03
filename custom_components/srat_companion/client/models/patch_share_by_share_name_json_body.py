from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.patch_share_by_share_name_json_body_usage import (
    PatchShareByShareNameJsonBodyUsage,
)
from ..types import UNSET, Unset

T = TypeVar("T", bound="PatchShareByShareNameJsonBody")


@_attrs_define
class PatchShareByShareNameJsonBody:
    """

    Attributes:
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.
        disabled (Union[Unset, bool]):
        ha_status (Union[Unset, str]):
        invalid (Union[Unset, bool]):
        is_ha_mounted (Union[Unset, bool]):
        mount_point_data (Union[Unset, Any]):
        name (Union[Unset, str]):
        recycle_bin_enabled (Union[Unset, bool]):
        ro_users (Union[Unset, list[Any]]):
        timemachine (Union[Unset, bool]):
        usage (Union[Unset, PatchShareByShareNameJsonBodyUsage]):
        users (Union[Unset, list[Any]]):
        veto_files (Union[Unset, list[str]]):

    """

    schema: Unset | str = UNSET
    disabled: Unset | bool = UNSET
    ha_status: Unset | str = UNSET
    invalid: Unset | bool = UNSET
    is_ha_mounted: Unset | bool = UNSET
    mount_point_data: Unset | Any = UNSET
    name: Unset | str = UNSET
    recycle_bin_enabled: Unset | bool = UNSET
    ro_users: Unset | list[Any] = UNSET
    timemachine: Unset | bool = UNSET
    usage: Unset | PatchShareByShareNameJsonBodyUsage = UNSET
    users: Unset | list[Any] = UNSET
    veto_files: Unset | list[str] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        disabled = self.disabled

        ha_status = self.ha_status

        invalid = self.invalid

        is_ha_mounted = self.is_ha_mounted

        mount_point_data = self.mount_point_data

        name = self.name

        recycle_bin_enabled = self.recycle_bin_enabled

        ro_users: Unset | list[Any] = UNSET
        if not isinstance(self.ro_users, Unset):
            ro_users = self.ro_users

        timemachine = self.timemachine

        usage: Unset | str = UNSET
        if not isinstance(self.usage, Unset):
            usage = self.usage.value

        users: Unset | list[Any] = UNSET
        if not isinstance(self.users, Unset):
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
        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        disabled = d.pop("disabled", UNSET)

        ha_status = d.pop("ha_status", UNSET)

        invalid = d.pop("invalid", UNSET)

        is_ha_mounted = d.pop("is_ha_mounted", UNSET)

        mount_point_data = d.pop("mount_point_data", UNSET)

        name = d.pop("name", UNSET)

        recycle_bin_enabled = d.pop("recycle_bin_enabled", UNSET)

        ro_users = cast("list[Any]", d.pop("ro_users", UNSET))

        timemachine = d.pop("timemachine", UNSET)

        _usage = d.pop("usage", UNSET)
        usage: Unset | PatchShareByShareNameJsonBodyUsage
        if isinstance(_usage, Unset):
            usage = UNSET
        else:
            usage = PatchShareByShareNameJsonBodyUsage(_usage)

        users = cast("list[Any]", d.pop("users", UNSET))

        veto_files = cast("list[str]", d.pop("veto_files", UNSET))

        patch_share_by_share_name_json_body = cls(
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

        patch_share_by_share_name_json_body.additional_properties = d
        return patch_share_by_share_name_json_body

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
