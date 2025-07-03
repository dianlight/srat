from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..models.json_patch_op_op import JsonPatchOpOp
from ..types import UNSET, Unset

T = TypeVar("T", bound="JsonPatchOp")


@_attrs_define
class JsonPatchOp:
    """

    Attributes:
        op (JsonPatchOpOp): Operation name
        path (str): JSON Pointer to the field being operated on, or the destination of a move/copy operation
        from_ (Union[Unset, str]): JSON Pointer for the source of a move or copy
        value (Union[Unset, Any]): The value to set

    """

    op: JsonPatchOpOp
    path: str
    from_: Unset | str = UNSET
    value: Unset | Any = UNSET

    def to_dict(self) -> dict[str, Any]:
        op = self.op.value

        path = self.path

        from_ = self.from_

        value = self.value

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "op": op,
                "path": path,
            }
        )
        if from_ is not UNSET:
            field_dict["from"] = from_
        if value is not UNSET:
            field_dict["value"] = value

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        op = JsonPatchOpOp(d.pop("op"))

        path = d.pop("path")

        from_ = d.pop("from", UNSET)

        value = d.pop("value", UNSET)

        json_patch_op = cls(
            op=op,
            path=path,
            from_=from_,
            value=value,
        )

        return json_patch_op
