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
    from ..models.samba_session_channels import SambaSessionChannels
    from ..models.samba_session_encryption_struct import SambaSessionEncryptionStruct
    from ..models.samba_session_signing_struct import SambaSessionSigningStruct


T = TypeVar("T", bound="SambaSession")


@_attrs_define
class SambaSession:
    """

    Attributes:
        auth_time (str):
        channels (SambaSessionChannels):
        creation_time (str):
        encryption (SambaSessionEncryptionStruct):
        gid (int):
        groupname (str):
        hostname (str):
        remote_machine (str):
        server_id (SambaServerID):
        session_dialect (str):
        session_id (str):
        signing (SambaSessionSigningStruct):
        uid (int):
        username (str):

    """

    auth_time: str
    channels: "SambaSessionChannels"
    creation_time: str
    encryption: "SambaSessionEncryptionStruct"
    gid: int
    groupname: str
    hostname: str
    remote_machine: str
    server_id: "SambaServerID"
    session_dialect: str
    session_id: str
    signing: "SambaSessionSigningStruct"
    uid: int
    username: str

    def to_dict(self) -> dict[str, Any]:
        auth_time = self.auth_time

        channels = self.channels.to_dict()

        creation_time = self.creation_time

        encryption = self.encryption.to_dict()

        gid = self.gid

        groupname = self.groupname

        hostname = self.hostname

        remote_machine = self.remote_machine

        server_id = self.server_id.to_dict()

        session_dialect = self.session_dialect

        session_id = self.session_id

        signing = self.signing.to_dict()

        uid = self.uid

        username = self.username

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "auth_time": auth_time,
                "channels": channels,
                "creation_time": creation_time,
                "encryption": encryption,
                "gid": gid,
                "groupname": groupname,
                "hostname": hostname,
                "remote_machine": remote_machine,
                "server_id": server_id,
                "session_dialect": session_dialect,
                "session_id": session_id,
                "signing": signing,
                "uid": uid,
                "username": username,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.samba_server_id import SambaServerID
        from ..models.samba_session_channels import SambaSessionChannels
        from ..models.samba_session_encryption_struct import (
            SambaSessionEncryptionStruct,
        )
        from ..models.samba_session_signing_struct import SambaSessionSigningStruct

        d = dict(src_dict)
        auth_time = d.pop("auth_time")

        channels = SambaSessionChannels.from_dict(d.pop("channels"))

        creation_time = d.pop("creation_time")

        encryption = SambaSessionEncryptionStruct.from_dict(d.pop("encryption"))

        gid = d.pop("gid")

        groupname = d.pop("groupname")

        hostname = d.pop("hostname")

        remote_machine = d.pop("remote_machine")

        server_id = SambaServerID.from_dict(d.pop("server_id"))

        session_dialect = d.pop("session_dialect")

        session_id = d.pop("session_id")

        signing = SambaSessionSigningStruct.from_dict(d.pop("signing"))

        uid = d.pop("uid")

        username = d.pop("username")

        samba_session = cls(
            auth_time=auth_time,
            channels=channels,
            creation_time=creation_time,
            encryption=encryption,
            gid=gid,
            groupname=groupname,
            hostname=hostname,
            remote_machine=remote_machine,
            server_id=server_id,
            session_dialect=session_dialect,
            session_id=session_id,
            signing=signing,
            uid=uid,
            username=username,
        )

        return samba_session
