from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
    Union,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.binary_asset import BinaryAsset


T = TypeVar("T", bound="ReleaseAsset")


@_attrs_define
class ReleaseAsset:
    """

    Attributes:
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.
        arch_asset (Union[Unset, BinaryAsset]):
        last_release (Union[Unset, str]):

    """

    schema: Unset | str = UNSET
    arch_asset: Union[Unset, "BinaryAsset"] = UNSET
    last_release: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        arch_asset: Unset | dict[str, Any] = UNSET
        if not isinstance(self.arch_asset, Unset):
            arch_asset = self.arch_asset.to_dict()

        last_release = self.last_release

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if arch_asset is not UNSET:
            field_dict["arch_asset"] = arch_asset
        if last_release is not UNSET:
            field_dict["last_release"] = last_release

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.binary_asset import BinaryAsset

        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        _arch_asset = d.pop("arch_asset", UNSET)
        arch_asset: Unset | BinaryAsset
        if isinstance(_arch_asset, Unset):
            arch_asset = UNSET
        else:
            arch_asset = BinaryAsset.from_dict(_arch_asset)

        last_release = d.pop("last_release", UNSET)

        release_asset = cls(
            schema=schema,
            arch_asset=arch_asset,
            last_release=last_release,
        )

        return release_asset
