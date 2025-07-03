from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Literal,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.update_progress import UpdateProgress


T = TypeVar("T", bound="SseEventUpdating")


@_attrs_define
class SseEventUpdating:
    """

    Attributes:
        data (UpdateProgress):
        event (Literal['updating']): The event name.
        id (Union[Unset, int]): The event ID.
        retry (Union[Unset, int]): The retry time in milliseconds.

    """

    data: "UpdateProgress"
    event: Literal["updating"]
    id: Unset | int = UNSET
    retry: Unset | int = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        data = self.data.to_dict()

        event = self.event

        id = self.id

        retry = self.retry

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "data": data,
                "event": event,
            }
        )
        if id is not UNSET:
            field_dict["id"] = id
        if retry is not UNSET:
            field_dict["retry"] = retry

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.update_progress import UpdateProgress

        d = dict(src_dict)
        data = UpdateProgress.from_dict(d.pop("data"))

        event = cast("Literal['updating']", d.pop("event"))
        if event != "updating":
            raise ValueError(f"event must match const 'updating', got '{event}'")

        id = d.pop("id", UNSET)

        retry = d.pop("retry", UNSET)

        sse_event_updating = cls(
            data=data,
            event=event,
            id=id,
            retry=retry,
        )

        sse_event_updating.additional_properties = d
        return sse_event_updating

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
