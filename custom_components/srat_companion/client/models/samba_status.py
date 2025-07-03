from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.samba_status_sessions import SambaStatusSessions
    from ..models.samba_status_tcons import SambaStatusTcons


T = TypeVar("T", bound="SambaStatus")


@_attrs_define
class SambaStatus:
    """

    Attributes:
        sessions (SambaStatusSessions):
        smb_conf (str):
        tcons (SambaStatusTcons):
        timestamp (str):
        version (str):
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.

    """

    sessions: "SambaStatusSessions"
    smb_conf: str
    tcons: "SambaStatusTcons"
    timestamp: str
    version: str
    schema: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        sessions = self.sessions.to_dict()

        smb_conf = self.smb_conf

        tcons = self.tcons.to_dict()

        timestamp = self.timestamp

        version = self.version

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "sessions": sessions,
                "smb_conf": smb_conf,
                "tcons": tcons,
                "timestamp": timestamp,
                "version": version,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.samba_status_sessions import SambaStatusSessions
        from ..models.samba_status_tcons import SambaStatusTcons

        d = dict(src_dict)
        sessions = SambaStatusSessions.from_dict(d.pop("sessions"))

        smb_conf = d.pop("smb_conf")

        tcons = SambaStatusTcons.from_dict(d.pop("tcons"))

        timestamp = d.pop("timestamp")

        version = d.pop("version")

        schema = d.pop("$schema", UNSET)

        samba_status = cls(
            sessions=sessions,
            smb_conf=smb_conf,
            tcons=tcons,
            timestamp=timestamp,
            version=version,
            schema=schema,
        )

        return samba_status
