from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...types import UNSET, Response, Unset


def _get_kwargs(
    mount_path_hash: str,
    *,
    force: Unset | bool = False,
    lazy: Unset | bool = False,
) -> dict[str, Any]:
    params: dict[str, Any] = {}

    params["force"] = force

    params["lazy"] = lazy

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": f"/volume/{mount_path_hash}/mount",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Any | None:
    if response.status_code == 204:
        return None
    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Any]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    mount_path_hash: str,
    *,
    client: AuthenticatedClient | Client,
    force: Unset | bool = False,
    lazy: Unset | bool = False,
) -> Response[Any]:
    """
    Delete volume by mount path hash mount

    Args:
        mount_path_hash (str):
        force (Union[Unset, bool]): Force umount operation Default: False.
        lazy (Union[Unset, bool]): Lazy umount operation Default: False.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any]

    """
    kwargs = _get_kwargs(
        mount_path_hash=mount_path_hash,
        force=force,
        lazy=lazy,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


async def asyncio_detailed(
    mount_path_hash: str,
    *,
    client: AuthenticatedClient | Client,
    force: Unset | bool = False,
    lazy: Unset | bool = False,
) -> Response[Any]:
    """
    Delete volume by mount path hash mount

    Args:
        mount_path_hash (str):
        force (Union[Unset, bool]): Force umount operation Default: False.
        lazy (Union[Unset, bool]): Lazy umount operation Default: False.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any]

    """
    kwargs = _get_kwargs(
        mount_path_hash=mount_path_hash,
        force=force,
        lazy=lazy,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)
