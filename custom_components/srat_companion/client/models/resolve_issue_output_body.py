from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="ResolveIssueOutputBody")


@_attrs_define
class ResolveIssueOutputBody:
    """

    Attributes:
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.

    """

    schema: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        resolve_issue_output_body = cls(
            schema=schema,
        )

        return resolve_issue_output_body
