import datetime
from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define
from dateutil.parser import isoparse

T = TypeVar("T", bound="ProcessStatus")


@_attrs_define
class ProcessStatus:
    """

    Attributes:
        connections (int):
        cpu_percent (float):
        create_time (datetime.datetime):
        is_running (bool):
        memory_percent (float):
        name (str):
        open_files (int):
        pid (int):
        status (Union[None, list[str]]):

    """

    connections: int
    cpu_percent: float
    create_time: datetime.datetime
    is_running: bool
    memory_percent: float
    name: str
    open_files: int
    pid: int
    status: None | list[str]

    def to_dict(self) -> dict[str, Any]:
        connections = self.connections

        cpu_percent = self.cpu_percent

        create_time = self.create_time.isoformat()

        is_running = self.is_running

        memory_percent = self.memory_percent

        name = self.name

        open_files = self.open_files

        pid = self.pid

        status: None | list[str]
        if isinstance(self.status, list):
            status = self.status

        else:
            status = self.status

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "connections": connections,
                "cpu_percent": cpu_percent,
                "create_time": create_time,
                "is_running": is_running,
                "memory_percent": memory_percent,
                "name": name,
                "open_files": open_files,
                "pid": pid,
                "status": status,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        connections = d.pop("connections")

        cpu_percent = d.pop("cpu_percent")

        create_time = isoparse(d.pop("create_time"))

        is_running = d.pop("is_running")

        memory_percent = d.pop("memory_percent")

        name = d.pop("name")

        open_files = d.pop("open_files")

        pid = d.pop("pid")

        def _parse_status(data: object) -> None | list[str]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                status_type_0 = cast("list[str]", data)

                return status_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[str]", data)

        status = _parse_status(d.pop("status"))

        process_status = cls(
            connections=connections,
            cpu_percent=cpu_percent,
            create_time=create_time,
            is_running=is_running,
            memory_percent=memory_percent,
            name=name,
            open_files=open_files,
            pid=pid,
            status=status,
        )

        return process_status
