from collections.abc import Mapping
from typing import Any, Self, TypeVar

from attrs import define as _attrs_define

T = TypeVar("T", bound="SambaServerID")


@_attrs_define
class SambaServerID:
    """

    Attributes:
        pid (str):
        task_id (str):
        unique_id (str):
        vnn (str):

    """

    pid: str
    task_id: str
    unique_id: str
    vnn: str

    def to_dict(self) -> dict[str, Any]:
        pid = self.pid

        task_id = self.task_id

        unique_id = self.unique_id

        vnn = self.vnn

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "pid": pid,
                "task_id": task_id,
                "unique_id": unique_id,
                "vnn": vnn,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        pid = d.pop("pid")

        task_id = d.pop("task_id")

        unique_id = d.pop("unique_id")

        vnn = d.pop("vnn")

        samba_server_id = cls(
            pid=pid,
            task_id=task_id,
            unique_id=unique_id,
            vnn=vnn,
        )

        return samba_server_id
