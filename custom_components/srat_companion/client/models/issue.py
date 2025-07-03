import datetime
from collections.abc import Mapping
from typing import (
    Any,
    Self,
    TypeVar,
)

from attrs import define as _attrs_define
from dateutil.parser import isoparse

from ..types import UNSET, Unset

T = TypeVar("T", bound="Issue")


@_attrs_define
class Issue:
    """

    Attributes:
        date (datetime.datetime):
        description (str):
        id (int):
        title (str):
        schema (Union[Unset, str]): A URL to the JSON Schema for this object.
        detail_link (Union[Unset, str]):
        resolution_link (Union[Unset, str]):

    """

    date: datetime.datetime
    description: str
    id: int
    title: str
    schema: Unset | str = UNSET
    detail_link: Unset | str = UNSET
    resolution_link: Unset | str = UNSET

    def to_dict(self) -> dict[str, Any]:
        date = self.date.isoformat()

        description = self.description

        id = self.id

        title = self.title

        schema = self.schema

        detail_link = self.detail_link

        resolution_link = self.resolution_link

        field_dict: dict[str, Any] = {}

        field_dict.update(
            {
                "date": date,
                "description": description,
                "id": id,
                "title": title,
            }
        )
        if schema is not UNSET:
            field_dict["$schema"] = schema
        if detail_link is not UNSET:
            field_dict["detailLink"] = detail_link
        if resolution_link is not UNSET:
            field_dict["resolutionLink"] = resolution_link

        return field_dict

    @classmethod
    def from_dict(cls, src_dict: Mapping[str, Any]) -> Self:
        d = dict(src_dict)
        date = isoparse(d.pop("date"))

        description = d.pop("description")

        id = d.pop("id")

        title = d.pop("title")

        schema = d.pop("$schema", UNSET)

        detail_link = d.pop("detailLink", UNSET)

        resolution_link = d.pop("resolutionLink", UNSET)

        issue = cls(
            date=date,
            description=description,
            id=id,
            title=title,
            schema=schema,
            detail_link=detail_link,
            resolution_link=resolution_link,
        )

        return issue
