from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

from ..models.welcome_supported_events import WelcomeSupportedEvents
from ..models.welcome_update_channel import WelcomeUpdateChannel

T = TypeVar("T", bound="Welcome")


@_attrs_define
class Welcome:
    """

    Attributes:
        message (str):
        supported_events (WelcomeSupportedEvents):
        update_channel (WelcomeUpdateChannel):

    """

    message: str
    supported_events: WelcomeSupportedEvents
    update_channel: WelcomeUpdateChannel

    def to_dict(self) -> dict[str, Any]:
        message = self.message

        supported_events = self.supported_events.value

        update_channel = self.update_channel.value

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "message": message,
                "supported_events": supported_events,
                "update_channel": update_channel,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        message = d.pop("message")

        supported_events = WelcomeSupportedEvents(d.pop("supported_events"))

        update_channel = WelcomeUpdateChannel(d.pop("update_channel"))

        welcome = cls(
            message=message,
            supported_events=supported_events,
            update_channel=update_channel,
        )

        return welcome
