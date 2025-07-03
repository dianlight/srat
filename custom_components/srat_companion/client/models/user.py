from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="User")


@_attrs_define
class User:
    """

    Attributes:
        username (str):
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.
        is_admin (Union[Unset, bool]):  Default: False.
        password (Union[Unset, str]):
        ro_shares (Union[None, Unset, list[str]]):
        rw_shares (Union[None, Unset, list[str]]):

    """

    username: str
    schema: Unset | str = UNSET
    is_admin: Unset | bool = False
    password: Unset | str = UNSET
    ro_shares: None | Unset | list[str] = UNSET
    rw_shares: None | Unset | list[str] = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        username = self.username

        schema = self.schema

        is_admin = self.is_admin

        password = self.password

        ro_shares: None | Unset | list[str]
        if isinstance(self.ro_shares, Unset):
            ro_shares = UNSET
        elif isinstance(self.ro_shares, list):
            ro_shares = self.ro_shares

        else:
            ro_shares = self.ro_shares

        rw_shares: None | Unset | list[str]
        if isinstance(self.rw_shares, Unset):
            rw_shares = UNSET
        elif isinstance(self.rw_shares, list):
            rw_shares = self.rw_shares

        else:
            rw_shares = self.rw_shares

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "username": username,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if is_admin is not UNSET:
            field_dict["is_admin"] = is_admin
        if password is not UNSET:
            field_dict["password"] = password
        if ro_shares is not UNSET:
            field_dict["ro_shares"] = ro_shares
        if rw_shares is not UNSET:
            field_dict["rw_shares"] = rw_shares

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        username = d.pop("username")

        schema = d.pop("$schema", UNSET)

        is_admin = d.pop("is_admin", UNSET)

        password = d.pop("password", UNSET)

        def _parse_ro_shares(data: object) -> None | Unset | list[str]:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                ro_shares_type_0 = cast("list[str]", data)

                return ro_shares_type_0
            except:  # noqa: E722
                pass
            return cast("None | Unset | list[str]", data)

        ro_shares = _parse_ro_shares(d.pop("ro_shares", UNSET))

        def _parse_rw_shares(data: object) -> None | Unset | list[str]:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                rw_shares_type_0 = cast("list[str]", data)

                return rw_shares_type_0
            except:  # noqa: E722
                pass
            return cast("None | Unset | list[str]", data)

        rw_shares = _parse_rw_shares(d.pop("rw_shares", UNSET))

        user = cls(
            username=username,
            schema=schema,
            is_admin=is_admin,
            password=password,
            ro_shares=ro_shares,
            rw_shares=rw_shares,
        )

        user.additional_properties = d
        return user

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
