from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.per_partition_info import PerPartitionInfo


T = TypeVar("T", bound="DiskHealthPerPartitionInfo")


@_attrs_define
class DiskHealthPerPartitionInfo:
    """ """

    additional_properties: dict[str, None | list["PerPartitionInfo"]] = _attrs_field(
        init=False, factory=dict
    )

    def to_dict(self) -> dict[str, Any]:
        field_dict: dict[str, Any] = {}
        for prop_name, prop in self.additional_properties.items():
            if isinstance(prop, list):
                field_dict[prop_name] = []
                for additional_property_type_0_item_data in prop:
                    additional_property_type_0_item = (
                        additional_property_type_0_item_data.to_dict()
                    )
                    field_dict[prop_name].append(additional_property_type_0_item)

            else:
                field_dict[prop_name] = prop

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.per_partition_info import PerPartitionInfo

        d = dict(src_dict)
        disk_health_per_partition_info = cls()

        additional_properties = {}
        for prop_name, prop_dict in d.items():

            def _parse_additional_property(
                data: object,
            ) -> None | list["PerPartitionInfo"]:
                if data is None:
                    return data
                try:
                    if not isinstance(data, list):
                        raise TypeError
                    additional_property_type_0 = []
                    _additional_property_type_0 = data
                    for (
                        additional_property_type_0_item_data
                    ) in _additional_property_type_0:
                        additional_property_type_0_item = PerPartitionInfo.from_dict(
                            additional_property_type_0_item_data
                        )

                        additional_property_type_0.append(
                            additional_property_type_0_item
                        )

                    return additional_property_type_0
                except:  # noqa: E722
                    pass
                return cast("None | list[PerPartitionInfo]", data)

            additional_property = _parse_additional_property(prop_dict)

            additional_properties[prop_name] = additional_property

        disk_health_per_partition_info.additional_properties = additional_properties
        return disk_health_per_partition_info

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> None | list["PerPartitionInfo"]:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: None | list["PerPartitionInfo"]) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
