from http import HTTPStatus
from typing import Any, Union

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.json_patch_op import JsonPatchOp
from ...models.patch_share_by_share_name_json_body import PatchShareByShareNameJsonBody
from ...models.shared_resource import SharedResource
from ...types import Response


def _get_kwargs(
    share_name: str,
    *,
    body: None | list["JsonPatchOp"] | PatchShareByShareNameJsonBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": f"/share/{share_name}",
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
    if isinstance(body, PatchShareByShareNameJsonBody):
        _kwargs["json"] = body.to_dict()

        headers["Content-Type"] = "application/merge-patch+json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> SharedResource | None:
    if response.status_code == 200:
        response_200 = SharedResource.from_dict(response.json())

        return response_200
    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[SharedResource]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    share_name: str,
    *,
    client: AuthenticatedClient | Client,
    body: None | list["JsonPatchOp"] | PatchShareByShareNameJsonBody,
) -> Response[SharedResource]:
    """
    Patch share-by-share-name

     Partial update operation supporting both JSON Merge Patch & JSON Patch updates.

    Args:
        share_name (str): Name of the share
        body (Union[None, list['JsonPatchOp']]):
        body (PatchShareByShareNameJsonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[SharedResource]

    """
    kwargs = _get_kwargs(
        share_name=share_name,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    share_name: str,
    *,
    client: AuthenticatedClient | Client,
    body: None | list["JsonPatchOp"] | PatchShareByShareNameJsonBody,
) -> SharedResource | None:
    """
    Patch share-by-share-name

     Partial update operation supporting both JSON Merge Patch & JSON Patch updates.

    Args:
        share_name (str): Name of the share
        body (Union[None, list['JsonPatchOp']]):
        body (PatchShareByShareNameJsonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        SharedResource

    """
    return sync_detailed(
        share_name=share_name,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    share_name: str,
    *,
    client: AuthenticatedClient | Client,
    body: None | list["JsonPatchOp"] | PatchShareByShareNameJsonBody,
) -> Response[SharedResource]:
    """
    Patch share-by-share-name

     Partial update operation supporting both JSON Merge Patch & JSON Patch updates.

    Args:
        share_name (str): Name of the share
        body (Union[None, list['JsonPatchOp']]):
        body (PatchShareByShareNameJsonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[SharedResource]

    """
    kwargs = _get_kwargs(
        share_name=share_name,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    share_name: str,
    *,
    client: AuthenticatedClient | Client,
    body: None | list["JsonPatchOp"] | PatchShareByShareNameJsonBody,
) -> SharedResource | None:
    """
    Patch share-by-share-name

     Partial update operation supporting both JSON Merge Patch & JSON Patch updates.

    Args:
        share_name (str): Name of the share
        body (Union[None, list['JsonPatchOp']]):
        body (PatchShareByShareNameJsonBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        SharedResource

    """
    return (
        await asyncio_detailed(
            share_name=share_name,
            client=client,
            body=body,
        )
    ).parsed
