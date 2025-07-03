from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="SmbConf")


@_attrs_define
class SmbConf:
    """

    Attributes:
        data (str):
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.

    """

    data: str
    schema: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        data = self.data

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "data": data,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        data = d.pop("data")

        schema = d.pop("$schema", UNSET)

        smb_conf = cls(
            data=data,
            schema=schema,
        )

        return smb_conf
