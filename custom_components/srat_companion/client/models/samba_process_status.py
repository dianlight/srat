from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

if TYPE_CHECKING:
    from ..models.process_status import ProcessStatus


T = TypeVar("T", bound="SambaProcessStatus")


@_attrs_define
class SambaProcessStatus:
    """

    Attributes:
        avahi (ProcessStatus):
        nmbd (ProcessStatus):
        smbd (ProcessStatus):
        wsdd2 (ProcessStatus):

    """

    avahi: "ProcessStatus"
    nmbd: "ProcessStatus"
    smbd: "ProcessStatus"
    wsdd2: "ProcessStatus"

    def to_dict(self) -> dict[str, Any]:
        avahi = self.avahi.to_dict()

        nmbd = self.nmbd.to_dict()

        smbd = self.smbd.to_dict()

        wsdd2 = self.wsdd2.to_dict()

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "avahi": avahi,
                "nmbd": nmbd,
                "smbd": smbd,
                "wsdd2": wsdd2,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.process_status import ProcessStatus

        d = dict(src_dict)
        avahi = ProcessStatus.from_dict(d.pop("avahi"))

        nmbd = ProcessStatus.from_dict(d.pop("nmbd"))

        smbd = ProcessStatus.from_dict(d.pop("smbd"))

        wsdd2 = ProcessStatus.from_dict(d.pop("wsdd2"))

        samba_process_status = cls(
            avahi=avahi,
            nmbd=nmbd,
            smbd=smbd,
            wsdd2=wsdd2,
        )

        return samba_process_status
