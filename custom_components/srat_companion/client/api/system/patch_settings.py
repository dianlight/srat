from http import HTTPStatus
from typing import Any, Union

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.json_patch_op import JsonPatchOp
from ...models.patch_settings_json_body import PatchSettingsJsonBody
from ...models.settings import Settings
from ...types import Response


def _get_kwargs(
    *,
    body: None | list["JsonPatchOp"] | PatchSettingsJsonBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/settings",
    }

    if isinstance(body, Union[None, list["JsonPatchOp"]]):
        _kwargs["json"]: None | list[dict[str, Any]]
        if isinstance(body, list):
            _kwargs["json"] = []
            for body_type_0_item_data in body:
                body_type_0_item = body_type_0_item_data.to_dict()
                _kwargs["json"].append(body_type_0_item)

        else:
            _kwargs["json"] = body

        headers["Content-Type"] = "application/json-patch+json"
    if isinstance(body, PatchSettingsJsonBody):
        _kwargs["json"] = body.to_dict()

        headers["Content-Type"] = "application/merge-patch+json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Settings | None:
    if response.status_code == 200:
        response_200 = Settings.from_dict(response.json())

        return response_200
    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Settings]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: None | list["JsonPatchOp"] | PatchSettingsJsonBody,
) -> Response[Settings]:
    """
    Patch settings

     Partial update operation supporting both JSON Merge Patch & JSON Patch updates.

    Args:
        body (Union[None, list['JsonPatchOp']]):
        body (PatchSettingsJsonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Settings]

    """
    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: None | list["JsonPatchOp"] | PatchSettingsJsonBody,
) -> Settings | None:
    """
    Patch settings

     Partial update operation supporting both JSON Merge Patch & JSON Patch updates.

    Args:
        body (Union[None, list['JsonPatchOp']]):
        body (PatchSettingsJsonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Settings

    """
    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: None | list["JsonPatchOp"] | PatchSettingsJsonBody,
) -> Response[Settings]:
    """
    Patch settings

     Partial update operation supporting both JSON Merge Patch & JSON Patch updates.

    Args:
        body (Union[None, list['JsonPatchOp']]):
        body (PatchSettingsJsonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Settings]

    """
    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: None | list["JsonPatchOp"] | PatchSettingsJsonBody,
) -> Settings | None:
    """
    Patch settings

     Partial update operation supporting both JSON Merge Patch & JSON Patch updates.

    Args:
        body (Union[None, list['JsonPatchOp']]):
        body (PatchSettingsJsonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Settings

    """
    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
