from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="DataDirtyTracker")


@_attrs_define
class DataDirtyTracker:
    """

    Attributes:
        settings (bool):
        shares (bool):
        users (bool):
        volumes (bool):

    """

    settings: bool
    shares: bool
    users: bool
    volumes: bool

    def to_dict(self) -> dict[str, Any]:
        settings = self.settings

        shares = self.shares

        users = self.users

        volumes = self.volumes

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "settings": settings,
                "shares": shares,
                "users": users,
                "volumes": volumes,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        settings = d.pop("settings")

        shares = d.pop("shares")

        users = d.pop("users")

        volumes = d.pop("volumes")

        data_dirty_tracker = cls(
            settings=settings,
            shares=shares,
            users=users,
            volumes=volumes,
        )

        return data_dirty_tracker
