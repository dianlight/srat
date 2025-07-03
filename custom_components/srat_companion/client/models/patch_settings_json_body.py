from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define

from ..models.patch_settings_json_body_update_channel import (
    PatchSettingsJsonBodyUpdateChannel,
)
from ..types import UNSET, Unset

T = TypeVar("T", bound="PatchSettingsJsonBody")


@_attrs_define
class PatchSettingsJsonBody:
    """

    Attributes:
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.
        allow_hosts (Union[Unset, list[str]]):
        bind_all_interfaces (Union[Unset, bool]):
        compatibility_mode (Union[Unset, bool]):
        hostname (Union[Unset, str]):
        interfaces (Union[Unset, list[str]]):
        log_level (Union[Unset, str]):
        mountoptions (Union[Unset, list[str]]):
        multi_channel (Union[Unset, bool]):
        update_channel (Union[Unset, PatchSettingsJsonBodyUpdateChannel]):
        workgroup (Union[Unset, str]):

    """

    schema: Unset | str = UNSET
    allow_hosts: Unset | list[str] = UNSET
    bind_all_interfaces: Unset | bool = UNSET
    compatibility_mode: Unset | bool = UNSET
    hostname: Unset | str = UNSET
    interfaces: Unset | list[str] = UNSET
    log_level: Unset | str = UNSET
    mountoptions: Unset | list[str] = UNSET
    multi_channel: Unset | bool = UNSET
    update_channel: Unset | PatchSettingsJsonBodyUpdateChannel = UNSET
    workgroup: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        allow_hosts: Unset | list[str] = UNSET
        if not isinstance(self.allow_hosts, Unset):
            allow_hosts = self.allow_hosts

        bind_all_interfaces = self.bind_all_interfaces

        compatibility_mode = self.compatibility_mode

        hostname = self.hostname

        interfaces: Unset | list[str] = UNSET
        if not isinstance(self.interfaces, Unset):
            interfaces = self.interfaces

        log_level = self.log_level

        mountoptions: Unset | list[str] = UNSET
        if not isinstance(self.mountoptions, Unset):
            mountoptions = self.mountoptions

        multi_channel = self.multi_channel

        update_channel: Unset | str = UNSET
        if not isinstance(self.update_channel, Unset):
            update_channel = self.update_channel.value

        workgroup = self.workgroup

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if allow_hosts is not UNSET:
            field_dict["allow_hosts"] = allow_hosts
        if bind_all_interfaces is not UNSET:
            field_dict["bind_all_interfaces"] = bind_all_interfaces
        if compatibility_mode is not UNSET:
            field_dict["compatibility_mode"] = compatibility_mode
        if hostname is not UNSET:
            field_dict["hostname"] = hostname
        if interfaces is not UNSET:
            field_dict["interfaces"] = interfaces
        if log_level is not UNSET:
            field_dict["log_level"] = log_level
        if mountoptions is not UNSET:
            field_dict["mountoptions"] = mountoptions
        if multi_channel is not UNSET:
            field_dict["multi_channel"] = multi_channel
        if update_channel is not UNSET:
            field_dict["update_channel"] = update_channel
        if workgroup is not UNSET:
            field_dict["workgroup"] = workgroup

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        allow_hosts = cast("list[str]", d.pop("allow_hosts", UNSET))

        bind_all_interfaces = d.pop("bind_all_interfaces", UNSET)

        compatibility_mode = d.pop("compatibility_mode", UNSET)

        hostname = d.pop("hostname", UNSET)

        interfaces = cast("list[str]", d.pop("interfaces", UNSET))

        log_level = d.pop("log_level", UNSET)

        mountoptions = cast("list[str]", d.pop("mountoptions", UNSET))

        multi_channel = d.pop("multi_channel", UNSET)

        _update_channel = d.pop("update_channel", UNSET)
        update_channel: Unset | PatchSettingsJsonBodyUpdateChannel
        if isinstance(_update_channel, Unset):
            update_channel = UNSET
        else:
            update_channel = PatchSettingsJsonBodyUpdateChannel(_update_channel)

        workgroup = d.pop("workgroup", UNSET)

        patch_settings_json_body = cls(
            schema=schema,
            allow_hosts=allow_hosts,
            bind_all_interfaces=bind_all_interfaces,
            compatibility_mode=compatibility_mode,
            hostname=hostname,
            interfaces=interfaces,
            log_level=log_level,
            mountoptions=mountoptions,
            multi_channel=multi_channel,
            update_channel=update_channel,
            workgroup=workgroup,
        )

        return patch_settings_json_body
