from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="InterfaceAddr")


@_attrs_define
class InterfaceAddr:
    """

    Attributes:
        addr (str):

    """

    addr: str

    def to_dict(self) -> dict[str, Any]:
        addr = self.addr

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "addr": addr,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        addr = d.pop("addr")

        interface_addr = cls(
            addr=addr,
        )

        return interface_addr
