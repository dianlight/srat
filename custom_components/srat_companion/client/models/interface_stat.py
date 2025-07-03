from collections.abc import Mapping
from typing import (
    TYPE_CHECKING,
    Any,
    Self,
    TypeVar,
    cast,
)

from attrs import define as _attrs_define

if TYPE_CHECKING:
    from ..models.interface_addr import InterfaceAddr


T = TypeVar("T", bound="InterfaceStat")


@_attrs_define
class InterfaceStat:
    """

    Attributes:
        addrs (Union[None, list['InterfaceAddr']]):
        flags (Union[None, list[str]]):
        hardware_addr (str):
        index (int):
        mtu (int):
        name (str):

    """

    addrs: None | list["InterfaceAddr"]
    flags: None | list[str]
    hardware_addr: str
    index: int
    mtu: int
    name: str

    def to_dict(self) -> dict[str, Any]:
        addrs: None | list[dict[str, Any]]
        if isinstance(self.addrs, list):
            addrs = []
            for addrs_type_0_item_data in self.addrs:
                addrs_type_0_item = addrs_type_0_item_data.to_dict()
                addrs.append(addrs_type_0_item)

        else:
            addrs = self.addrs

        flags: None | list[str]
        if isinstance(self.flags, list):
            flags = self.flags

        else:
            flags = self.flags

        hardware_addr = self.hardware_addr

        index = self.index

        mtu = self.mtu

        name = self.name

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "addrs": addrs,
                "flags": flags,
                "hardwareAddr": hardware_addr,
                "index": index,
                "mtu": mtu,
                "name": name,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        from ..models.interface_addr import InterfaceAddr

        d = dict(src_dict)

        def _parse_addrs(data: object) -> None | list["InterfaceAddr"]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                addrs_type_0 = []
                _addrs_type_0 = data
                for addrs_type_0_item_data in _addrs_type_0:
                    addrs_type_0_item = InterfaceAddr.from_dict(addrs_type_0_item_data)

                    addrs_type_0.append(addrs_type_0_item)

                return addrs_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[InterfaceAddr]", data)

        addrs = _parse_addrs(d.pop("addrs"))

        def _parse_flags(data: object) -> None | list[str]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                flags_type_0 = cast("list[str]", data)

                return flags_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[str]", data)

        flags = _parse_flags(d.pop("flags"))

        hardware_addr = d.pop("hardwareAddr")

        index = d.pop("index")

        mtu = d.pop("mtu")

        name = d.pop("name")

        interface_stat = cls(
            addrs=addrs,
            flags=flags,
            hardware_addr=hardware_addr,
            index=index,
            mtu=mtu,
            name=name,
        )

        return interface_stat
