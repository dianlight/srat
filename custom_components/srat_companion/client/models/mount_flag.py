from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="MountFlag")


@_attrs_define
class MountFlag:
    """

    Attributes:
        name (str):
        description (Union[Unset, str]):
        needs_value (Union[Unset, bool]):
        value (Union[Unset, str]):
        value_description (Union[Unset, str]):
        value_validation_regex (Union[Unset, str]):

    """

    name: str
    description: Unset | str = UNSET
    needs_value: Unset | bool = UNSET
    value: Unset | str = UNSET
    value_description: Unset | str = UNSET
    value_validation_regex: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        description = self.description

        needs_value = self.needs_value

        value = self.value

        value_description = self.value_description

        value_validation_regex = self.value_validation_regex

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "name": name,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description
        if needs_value is not UNSET:
            field_dict["needsValue"] = needs_value
        if value is not UNSET:
            field_dict["value"] = value
        if value_description is not UNSET:
            field_dict["value_description"] = value_description
        if value_validation_regex is not UNSET:
            field_dict["value_validation_regex"] = value_validation_regex

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        name = d.pop("name")

        description = d.pop("description", UNSET)

        needs_value = d.pop("needsValue", UNSET)

        value = d.pop("value", UNSET)

        value_description = d.pop("value_description", UNSET)

        value_validation_regex = d.pop("value_validation_regex", UNSET)

        mount_flag = cls(
            name=name,
            description=description,
            needs_value=needs_value,
            value=value,
            value_description=value_description,
            value_validation_regex=value_validation_regex,
        )

        return mount_flag
