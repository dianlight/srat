from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..models.update_progress_update_process_state import (
    UpdateProgressUpdateProcessState,
)
from ..types import UNSET, Unset

T = TypeVar("T", bound="UpdateProgress")


@_attrs_define
class UpdateProgress:
    """

    Attributes:
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.
        error_message (Union[Unset, str]):
        last_release (Union[Unset, str]):
        progress (Union[Unset, int]):
        update_process_state (Union[Unset, UpdateProgressUpdateProcessState]):

    """

    schema: Unset | str = UNSET
    error_message: Unset | str = UNSET
    last_release: Unset | str = UNSET
    progress: Unset | int = UNSET
    update_process_state: Unset | UpdateProgressUpdateProcessState = UNSET

    def to_dict(self) -> dict[str, Any]:
        schema = self.schema

        error_message = self.error_message

        last_release = self.last_release

        progress = self.progress

        update_process_state: Unset | str = UNSET
        if not isinstance(self.update_process_state, Unset):
            update_process_state = self.update_process_state.value

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if error_message is not UNSET:
            field_dict["error_message"] = error_message
        if last_release is not UNSET:
            field_dict["last_release"] = last_release
        if progress is not UNSET:
            field_dict["progress"] = progress
        if update_process_state is not UNSET:
            field_dict["update_process_state"] = update_process_state

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        schema = d.pop("$schema", UNSET)

        error_message = d.pop("error_message", UNSET)

        last_release = d.pop("last_release", UNSET)

        progress = d.pop("progress", UNSET)

        _update_process_state = d.pop("update_process_state", UNSET)
        update_process_state: Unset | UpdateProgressUpdateProcessState
        if isinstance(_update_process_state, Unset):
            update_process_state = UNSET
        else:
            update_process_state = UpdateProgressUpdateProcessState(
                _update_process_state
            )

        update_progress = cls(
            schema=schema,
            error_message=error_message,
            last_release=last_release,
            progress=progress,
            update_process_state=update_process_state,
        )

        return update_progress
