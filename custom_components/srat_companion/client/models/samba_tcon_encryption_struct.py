from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="SambaTconEncryptionStruct")


@_attrs_define
class SambaTconEncryptionStruct:
    """

    Attributes:
        cipher (str):
        degree (str):

    """

    cipher: str
    degree: str

    def to_dict(self) -> dict[str, Any]:
        cipher = self.cipher

        degree = self.degree

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "cipher": cipher,
                "degree": degree,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        cipher = d.pop("cipher")

        degree = d.pop("degree")

        samba_tcon_encryption_struct = cls(
            cipher=cipher,
            degree=degree,
        )

        return samba_tcon_encryption_struct
