from http import HTTPStatus
from typing import Any, Union

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.sse_event_heartbeat import SseEventHeartbeat
from ...models.sse_event_hello import SseEventHello
from ...models.sse_event_share import SseEventShare
from ...models.sse_event_updating import SseEventUpdating
from ...models.sse_event_volumes import SseEventVolumes
from ...types import Response


def _get_kwargs() -> dict[str, Any]:
    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/sse",
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> (
    list[
        Union[
            "SseEventHeartbeat",
            "SseEventHello",
            "SseEventShare",
            "SseEventUpdating",
            "SseEventVolumes",
        ]
    ]
    | None
):
    if response.status_code == 200:
        response_200 = []
        _response_200 = response.text
        for response_200_item_data in _response_200:

            def _parse_response_200_item(
                data: object,
            ) -> Union[
                "SseEventHeartbeat",
                "SseEventHello",
                "SseEventShare",
                "SseEventUpdating",
                "SseEventVolumes",
            ]:
                try:
                    if not isinstance(data, dict):
                        raise TypeError
                    response_200_item_type_0 = SseEventHeartbeat.from_dict(data)

                    return response_200_item_type_0
                except:  # noqa: E722
                    pass
                try:
                    if not isinstance(data, dict):
                        raise TypeError
                    response_200_item_type_1 = SseEventHello.from_dict(data)

                    return response_200_item_type_1
                except:  # noqa: E722
                    pass
                try:
                    if not isinstance(data, dict):
                        raise TypeError
                    response_200_item_type_2 = SseEventShare.from_dict(data)

                    return response_200_item_type_2
                except:  # noqa: E722
                    pass
                try:
                    if not isinstance(data, dict):
                        raise TypeError
                    response_200_item_type_3 = SseEventUpdating.from_dict(data)

                    return response_200_item_type_3
                except:  # noqa: E722
                    pass
                if not isinstance(data, dict):
                    raise TypeError
                response_200_item_type_4 = SseEventVolumes.from_dict(data)

                return response_200_item_type_4

            response_200_item = _parse_response_200_item(response_200_item_data)

            response_200.append(response_200_item)

        return response_200
    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[
    list[
        Union[
            "SseEventHeartbeat",
            "SseEventHello",
            "SseEventShare",
            "SseEventUpdating",
            "SseEventVolumes",
        ]
    ]
]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[
    list[
        Union[
            "SseEventHeartbeat",
            "SseEventHello",
            "SseEventShare",
            "SseEventUpdating",
            "SseEventVolumes",
        ]
    ]
]:
    """
    Server sent events

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[list[Union['SseEventHeartbeat', 'SseEventHello', 'SseEventShare', 'SseEventUpdating', 'SseEventVolumes']]]

    """
    kwargs = _get_kwargs()

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
) -> (
    list[
        Union[
            "SseEventHeartbeat",
            "SseEventHello",
            "SseEventShare",
            "SseEventUpdating",
            "SseEventVolumes",
        ]
    ]
    | None
):
    """
    Server sent events

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        list[Union['SseEventHeartbeat', 'SseEventHello', 'SseEventShare', 'SseEventUpdating', 'SseEventVolumes']]

    """
    return sync_detailed(
        client=client,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[
    list[
        Union[
            "SseEventHeartbeat",
            "SseEventHello",
            "SseEventShare",
            "SseEventUpdating",
            "SseEventVolumes",
        ]
    ]
]:
    """
    Server sent events

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[list[Union['SseEventHeartbeat', 'SseEventHello', 'SseEventShare', 'SseEventUpdating', 'SseEventVolumes']]]

    """
    kwargs = _get_kwargs()

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
) -> (
    list[
        Union[
            "SseEventHeartbeat",
            "SseEventHello",
            "SseEventShare",
            "SseEventUpdating",
            "SseEventVolumes",
        ]
    ]
    | None
):
    """
    Server sent events

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        list[Union['SseEventHeartbeat', 'SseEventHello', 'SseEventShare', 'SseEventUpdating', 'SseEventVolumes']]

    """
    return (
        await asyncio_detailed(
            client=client,
        )
    ).parsed
