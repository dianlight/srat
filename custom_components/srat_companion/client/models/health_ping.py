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
    from ..models.addon_stats_data import AddonStatsData
    from ..models.data_dirty_tracker import DataDirtyTracker
    from ..models.disk_health import DiskHealth
    from ..models.network_stats import NetworkStats
    from ..models.release_asset import ReleaseAsset
    from ..models.samba_process_status import SambaProcessStatus
    from ..models.samba_status import SambaStatus


T = TypeVar("T", bound="HealthPing")


@_attrs_define
class HealthPing:
    """

    Attributes:
        addon_stats (AddonStatsData):
        alive (bool):
        alive_time (int):
        build_version (str):
        dirty_tracking (DataDirtyTracker):
        disk_health (DiskHealth):
        last_error (str):
        last_release (ReleaseAsset):
        network_health (NetworkStats):
        protected_mode (bool):
        read_only (bool):
        samba_process_status (SambaProcessStatus):
        samba_status (SambaStatus):
        secure_mode (bool):
        start_time (int):
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.

    """

    addon_stats: "AddonStatsData"
    alive: bool
    alive_time: int
    build_version: str
    dirty_tracking: "DataDirtyTracker"
    disk_health: "DiskHealth"
    last_error: str
    last_release: "ReleaseAsset"
    network_health: "NetworkStats"
    protected_mode: bool
    read_only: bool
    samba_process_status: "SambaProcessStatus"
    samba_status: "SambaStatus"
    secure_mode: bool
    start_time: int
    schema: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        addon_stats = self.addon_stats.to_dict()

        alive = self.alive

        alive_time = self.alive_time

        build_version = self.build_version

        dirty_tracking = self.dirty_tracking.to_dict()

        disk_health = self.disk_health.to_dict()

        last_error = self.last_error

        last_release = self.last_release.to_dict()

        network_health = self.network_health.to_dict()

        protected_mode = self.protected_mode

        read_only = self.read_only

        samba_process_status = self.samba_process_status.to_dict()

        samba_status = self.samba_status.to_dict()

        secure_mode = self.secure_mode

        start_time = self.start_time

        schema = self.schema

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "addon_stats": addon_stats,
                "alive": alive,
                "aliveTime": alive_time,
                "build_version": build_version,
                "dirty_tracking": dirty_tracking,
                "disk_health": disk_health,
                "last_error": last_error,
                "last_release": last_release,
                "network_health": network_health,
                "protected_mode": protected_mode,
                "read_only": read_only,
                "samba_process_status": samba_process_status,
                "samba_status": samba_status,
                "secure_mode": secure_mode,
                "startTime": start_time,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.addon_stats_data import AddonStatsData
        from ..models.data_dirty_tracker import DataDirtyTracker
        from ..models.disk_health import DiskHealth
        from ..models.network_stats import NetworkStats
        from ..models.release_asset import ReleaseAsset
        from ..models.samba_process_status import SambaProcessStatus
        from ..models.samba_status import SambaStatus

        d = dict(src_dict)
        addon_stats = AddonStatsData.from_dict(d.pop("addon_stats"))

        alive = d.pop("alive")

        alive_time = d.pop("aliveTime")

        build_version = d.pop("build_version")

        dirty_tracking = DataDirtyTracker.from_dict(d.pop("dirty_tracking"))

        disk_health = DiskHealth.from_dict(d.pop("disk_health"))

        last_error = d.pop("last_error")

        last_release = ReleaseAsset.from_dict(d.pop("last_release"))

        network_health = NetworkStats.from_dict(d.pop("network_health"))

        protected_mode = d.pop("protected_mode")

        read_only = d.pop("read_only")

        samba_process_status = SambaProcessStatus.from_dict(
            d.pop("samba_process_status")
        )

        samba_status = SambaStatus.from_dict(d.pop("samba_status"))

        secure_mode = d.pop("secure_mode")

        start_time = d.pop("startTime")

        schema = d.pop("$schema", UNSET)

        health_ping = cls(
            addon_stats=addon_stats,
            alive=alive,
            alive_time=alive_time,
            build_version=build_version,
            dirty_tracking=dirty_tracking,
            disk_health=disk_health,
            last_error=last_error,
            last_release=last_release,
            network_health=network_health,
            protected_mode=protected_mode,
            read_only=read_only,
            samba_process_status=samba_process_status,
            samba_status=samba_status,
            secure_mode=secure_mode,
            start_time=start_time,
            schema=schema,
        )

        return health_ping
