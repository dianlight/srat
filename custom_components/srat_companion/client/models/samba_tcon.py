from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define

if TYPE_CHECKING:
    from ..models.samba_server_id import SambaServerID
    from ..models.samba_tcon_encryption_struct import SambaTconEncryptionStruct
    from ..models.samba_tcon_signing_struct import SambaTconSigningStruct


T = TypeVar("T", bound="SambaTcon")


@_attrs_define
class SambaTcon:
    """

    Attributes:
        connected_at (str):
        device (str):
        encryption (SambaTconEncryptionStruct):
        machine (str):
        server_id (SambaServerID):
        service (str):
        session_id (str):
        share (str):
        signing (SambaTconSigningStruct):
        tcon_id (str):

    """

    connected_at: str
    device: str
    encryption: "SambaTconEncryptionStruct"
    machine: str
    server_id: "SambaServerID"
    service: str
    session_id: str
    share: str
    signing: "SambaTconSigningStruct"
    tcon_id: str

    def to_dict(self) -> dict[str, Any]:
        connected_at = self.connected_at

        device = self.device

        encryption = self.encryption.to_dict()

        machine = self.machine

        server_id = self.server_id.to_dict()

        service = self.service

        session_id = self.session_id

        share = self.share

        signing = self.signing.to_dict()

        tcon_id = self.tcon_id

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "connected_at": connected_at,
                "device": device,
                "encryption": encryption,
                "machine": machine,
                "server_id": server_id,
                "service": service,
                "session_id": session_id,
                "share": share,
                "signing": signing,
                "tcon_id": tcon_id,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.samba_server_id import SambaServerID
        from ..models.samba_tcon_encryption_struct import SambaTconEncryptionStruct
        from ..models.samba_tcon_signing_struct import SambaTconSigningStruct

        d = dict(src_dict)
        connected_at = d.pop("connected_at")

        device = d.pop("device")

        encryption = SambaTconEncryptionStruct.from_dict(d.pop("encryption"))

        machine = d.pop("machine")

        server_id = SambaServerID.from_dict(d.pop("server_id"))

        service = d.pop("service")

        session_id = d.pop("session_id")

        share = d.pop("share")

        signing = SambaTconSigningStruct.from_dict(d.pop("signing"))

        tcon_id = d.pop("tcon_id")

        samba_tcon = cls(
            connected_at=connected_at,
            device=device,
            encryption=encryption,
            machine=machine,
            server_id=server_id,
            service=service,
            session_id=session_id,
            share=share,
            signing=signing,
            tcon_id=tcon_id,
        )

        return samba_tcon
