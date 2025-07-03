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
    from ..models.partition import Partition


T = TypeVar("T", bound="Disk")


@_attrs_define
class Disk:
    """

    Attributes:
        connection_bus (Union[Unset, str]):
        device (Union[Unset, str]):
        ejectable (Union[Unset, bool]):
        id (Union[Unset, str]):
        model (Union[Unset, str]):
        partitions (Union[Unset, list['Partition']]):
        removable (Union[Unset, bool]):
        revision (Union[Unset, str]):
        seat (Union[Unset, str]):
        serial (Union[Unset, str]):
        size (Union[Unset, int]):
        vendor (Union[Unset, str]):

    """

    connection_bus: Unset | str = UNSET
    device: Unset | str = UNSET
    ejectable: Unset | bool = UNSET
    id: Unset | str = UNSET
    model: Unset | str = UNSET
    partitions: Unset | list["Partition"] = UNSET
    removable: Unset | bool = UNSET
    revision: Unset | str = UNSET
    seat: Unset | str = UNSET
    serial: Unset | str = UNSET
    size: Unset | int = UNSET
    vendor: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        connection_bus = self.connection_bus

        device = self.device

        ejectable = self.ejectable

        id = self.id

        model = self.model

        partitions: Unset | list[dict[str, Any]] = UNSET
        if not isinstance(self.partitions, Unset):
            partitions = []
            for partitions_item_data in self.partitions:
                partitions_item = partitions_item_data.to_dict()
                partitions.append(partitions_item)

        removable = self.removable

        revision = self.revision

        seat = self.seat

        serial = self.serial

        size = self.size

        vendor = self.vendor

        field_dict: dict[str, Any] = {}

        field_dict.update({})
        if connection_bus is not UNSET:
            field_dict["connection_bus"] = connection_bus
        if device is not UNSET:
            field_dict["device"] = device
        if ejectable is not UNSET:
            field_dict["ejectable"] = ejectable
        if id is not UNSET:
            field_dict["id"] = id
        if model is not UNSET:
            field_dict["model"] = model
        if partitions is not UNSET:
            field_dict["partitions"] = partitions
        if removable is not UNSET:
            field_dict["removable"] = removable
        if revision is not UNSET:
            field_dict["revision"] = revision
        if seat is not UNSET:
            field_dict["seat"] = seat
        if serial is not UNSET:
            field_dict["serial"] = serial
        if size is not UNSET:
            field_dict["size"] = size
        if vendor is not UNSET:
            field_dict["vendor"] = vendor

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.partition import Partition

        d = dict(src_dict)
        connection_bus = d.pop("connection_bus", UNSET)

        device = d.pop("device", UNSET)

        ejectable = d.pop("ejectable", UNSET)

        id = d.pop("id", UNSET)

        model = d.pop("model", UNSET)

        partitions = []
        _partitions = d.pop("partitions", UNSET)
        for partitions_item_data in _partitions or []:
            partitions_item = Partition.from_dict(partitions_item_data)

            partitions.append(partitions_item)

        removable = d.pop("removable", UNSET)

        revision = d.pop("revision", UNSET)

        seat = d.pop("seat", UNSET)

        serial = d.pop("serial", UNSET)

        size = d.pop("size", UNSET)

        vendor = d.pop("vendor", UNSET)

        disk = cls(
            connection_bus=connection_bus,
            device=device,
            ejectable=ejectable,
            id=id,
            model=model,
            partitions=partitions,
            removable=removable,
            revision=revision,
            seat=seat,
            serial=serial,
            size=size,
            vendor=vendor,
        )

        return disk
