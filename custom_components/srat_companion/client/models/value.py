from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="Value")


@_attrs_define
class Value:
    """

    Attributes:
        channel_id (str):
        creation_time (str):
        local_address (str):
        remote_address (str):

    """

    channel_id: str
    creation_time: str
    local_address: str
    remote_address: str

    def to_dict(self) -> dict[str, Any]:
        channel_id = self.channel_id

        creation_time = self.creation_time

        local_address = self.local_address

        remote_address = self.remote_address

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "channel_id": channel_id,
                "creation_time": creation_time,
                "local_address": local_address,
                "remote_address": remote_address,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        channel_id = d.pop("channel_id")

        creation_time = d.pop("creation_time")

        local_address = d.pop("local_address")

        remote_address = d.pop("remote_address")

        value = cls(
            channel_id=channel_id,
            creation_time=creation_time,
            local_address=local_address,
            remote_address=remote_address,
        )

        return value
