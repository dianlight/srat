from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

T = TypeVar("T", bound="BinaryAsset")


@_attrs_define
class BinaryAsset:
    """

    Attributes:
        id (int):
        name (str):
        size (int):
        browser_download_url (Union[Unset, str]):

    """

    id: int
    name: str
    size: int
    browser_download_url: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        size = self.size

        browser_download_url = self.browser_download_url

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "id": id,
                "name": name,
                "size": size,
            }
        )
        if browser_download_url is not UNSET:
            field_dict["browser_download_url"] = browser_download_url

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        size = d.pop("size")

        browser_download_url = d.pop("browser_download_url", UNSET)

        binary_asset = cls(
            id=id,
            name=name,
            size=size,
            browser_download_url=browser_download_url,
        )

        return binary_asset
