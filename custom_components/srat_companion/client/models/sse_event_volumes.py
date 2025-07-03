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
    from ..models.disk import Disk


T = TypeVar("T", bound="SseEventVolumes")


@_attrs_define
class SseEventVolumes:
    """

    Attributes:
        data (Union[None, list['Disk']]):
        event (Literal['volumes']): The event name.
        id (Union[Unset, int]): The event ID.
        retry (Union[Unset, int]): The retry time in milliseconds.

    """

    data: None | list["Disk"]
    event: Literal["volumes"]
    id: Unset | int = UNSET
    retry: Unset | int = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        data: None | list[dict[str, Any]]
        if isinstance(self.data, list):
            data = []
            for data_type_0_item_data in self.data:
                data_type_0_item = data_type_0_item_data.to_dict()
                data.append(data_type_0_item)

        else:
            data = self.data

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
        from ..models.disk import Disk

        d = dict(src_dict)

        def _parse_data(data: object) -> None | list["Disk"]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                data_type_0 = []
                _data_type_0 = data
                for data_type_0_item_data in _data_type_0:
                    data_type_0_item = Disk.from_dict(data_type_0_item_data)

                    data_type_0.append(data_type_0_item)

                return data_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[Disk]", data)

        data = _parse_data(d.pop("data"))

        event = cast("Literal['volumes']", d.pop("event"))
        if event != "volumes":
            raise ValueError(f"event must match const 'volumes', got '{event}'")

        id = d.pop("id", UNSET)

        retry = d.pop("retry", UNSET)

        sse_event_volumes = cls(
            data=data,
            event=event,
            id=id,
            retry=retry,
        )

        sse_event_volumes.additional_properties = d
        return sse_event_volumes

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
