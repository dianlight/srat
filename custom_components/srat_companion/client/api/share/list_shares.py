from http import HTTPStatus
from typing import Any, cast

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.shared_resource import SharedResource
from ...types import Response


def _get_kwargs() -> dict[str, Any]:
    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/shares",
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> None | list["SharedResource"]:
    if response.status_code == 200:

        def _parse_response_200(data: object) -> None | list["SharedResource"]:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError
                response_200_type_0 = []
                _response_200_type_0 = data
                for response_200_type_0_item_data in _response_200_type_0:
                    response_200_type_0_item = SharedResource.from_dict(
                        response_200_type_0_item_data
                    )

                    response_200_type_0.append(response_200_type_0_item)

                return response_200_type_0
            except:  # noqa: E722
                pass
            return cast("None | list[SharedResource]", data)

        response_200 = _parse_response_200(response.json())

        return response_200
    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[None | list["SharedResource"]]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[None | list["SharedResource"]]:
    """
    List shares

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Union[None, list['SharedResource']]]

    """
    kwargs = _get_kwargs()

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
) -> None | list["SharedResource"]:
    """
    List shares

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Union[None, list['SharedResource']]

    """
    return sync_detailed(
        client=client,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[None | list["SharedResource"]]:
    """
    List shares

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Union[None, list['SharedResource']]]

    """
    kwargs = _get_kwargs()

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
) -> None | list["SharedResource"]:
    """
    List shares

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Union[None, list['SharedResource']]

    """
    return (
        await asyncio_detailed(
            client=client,
        )
    ).parsed
