"""Helper to reuse the Home Assistant Dropbox integration token (task 050).

Reads ``hass.config_entries`` for domain ``dropbox`` (core integration since
2026.4) and forwards the OAuth token to the SRAT backend over the existing
WebSocket, so rclone can be authorized without a second browser flow.

Home Assistant parallel (Dropbox):
HA's Dropbox integration does NOT do pure client OAuth. By default it uses
Home Assistant Cloud Account Linking (Nabu Casa proxy at accounts.home-
assistant.io) which holds the app credentials; the browser fans out via
``https://my.home-assistant.io/redirect/oauth`` or a Nabu Casa cloudhook.
The Application Credentials path (user-created app) is the bring-your-own-app
alternative — identical to SRAT's custom_app mode. Third-party addons cannot
reuse HA's credentials, hence this file reuses the *token* already stored in
``hass.config_entries`` instead.
"""

from __future__ import annotations

import datetime
import json
import logging
from typing import Any

from homeassistant.core import HomeAssistant

_LOGGER = logging.getLogger(__name__)

_DROPBOX_DOMAIN = "dropbox"


def _token_to_rclone_json(token: dict[str, Any]) -> str:
    """Convert HA OAuth token dict to rclone Dropbox token JSON.

    HA stores ``token`` as returned by the OAuth2 session helper, typically
    ``{"access_token": "...", "refresh_token": "...", "expires_in": 14400,
    "expires_at": 1234567890.0, "token_type": "bearer"}`` (shape varies
    slightly between Cloud Linking and custom-app entries). Rclone's Dropbox
    driver expects ``{"access_token": ..., "refresh_token": ..., "expiry":
    "RFC3339"}`` together with optional ``token_type``/``expires_in``.
    """
    out: dict[str, Any] = {}
    # Copy known fields verbatim when present.
    for key in ("access_token", "refresh_token", "token_type", "expires_in", "scope"):
        if key in token:
            out[key] = token[key]
    # HA uses ``expires_at`` (unix timestamp float); rclone uses ``expiry``
    # (RFC3339 string). Prefer an explicit ``expiry`` if already present.
    if "expiry" in token and isinstance(token["expiry"], str) and token["expiry"]:
        out["expiry"] = token["expiry"]
    elif "expires_at" in token:
        try:
            ts = float(token["expires_at"])
            dt = datetime.datetime.fromtimestamp(ts, tz=datetime.UTC)
            out["expiry"] = dt.isoformat().replace("+00:00", "Z")
            # Also keep expires_in when available for completeness.
            if "expires_in" not in out and "expires_in" in token:
                out["expires_in"] = token["expires_in"]
        except (ValueError, TypeError, OSError, OverflowError):
            _LOGGER.debug(
                "Failed to convert expires_at %r to expiry", token.get("expires_at")
            )
    elif "expires_in" in token:
        # No absolute timestamp — synthesize expiry from now + expires_in.
        try:
            seconds = int(token["expires_in"])
            dt = datetime.datetime.now(tz=datetime.UTC) + datetime.timedelta(
                seconds=seconds
            )
            out["expiry"] = dt.isoformat().replace("+00:00", "Z")
        except (ValueError, TypeError):
            pass
    # Fallback: if we have no expiry at all, rclone will treat the token as
    # expiring immediately and trigger a refresh on first use (which will fail
    # without client credentials for Cloud Linking entries — accepted trade-off
    # documented in task 049/050).
    return json.dumps(out)


def _extract_token(entry_data: dict[str, Any]) -> dict[str, Any] | None:
    """Extract OAuth token dict from a Dropbox config entry data blob."""
    # Core dropbox stores the OAuth token under ``token`` (OAuth2Session).
    # Be tolerant of historical shapes.
    token = entry_data.get("token")
    if isinstance(token, dict) and token.get("access_token"):
        return token
    # Fallback: some forks stored it flat.
    if isinstance(entry_data.get("access_token"), str):
        return {
            "access_token": entry_data.get("access_token"),
            "refresh_token": entry_data.get("refresh_token", ""),
            "expires_at": entry_data.get("expires_at"),
            "expires_in": entry_data.get("expires_in"),
            "token_type": entry_data.get("token_type", "bearer"),
        }
    return None


def _extract_client_credentials(entry_data: dict[str, Any]) -> tuple[str, str]:
    """Best-effort extraction of client_id/secret for refresh binding.

    For Application Credentials entries the credentials live in the
    ``application_credentials`` storage, not in the entry data itself. We
    attempt to read ``client_id``/``client_secret`` when they are present
    (e.g. older manual entries or non-standard forks); otherwise return empty
    strings, which the backend treats as Cloud Linking mode (trade-off: rclone
    cannot refresh without the Nabu Casa app secret, but the current token
    still works until expiry).
    """
    client_id = entry_data.get("client_id", "") or entry_data.get("clientId", "")
    client_secret = entry_data.get("client_secret", "") or entry_data.get(
        "clientSecret", ""
    )
    if isinstance(client_id, str) and isinstance(client_secret, str):
        return client_id.strip(), client_secret.strip()
    return "", ""


def get_dropbox_token_payload(hass: HomeAssistant) -> dict[str, Any] | None:
    """Return the best Dropbox token payload to forward, or None if absent.

    Picks the first loaded Dropbox entry with a usable token. If multiple
    entries exist, the first one wins (multi-account is not a SRAT use-case
    today; the user can pick the desired account by ensuring a single entry).
    """
    try:
        entries = hass.config_entries.async_entries(_DROPBOX_DOMAIN)
    except Exception:
        _LOGGER.debug("Failed to enumerate dropbox config entries", exc_info=True)
        return None

    for entry in entries:
        try:
            data = dict(entry.data) if isinstance(entry.data, dict) else {}
        except Exception:
            _LOGGER.debug("Failed to read dropbox entry data", exc_info=True)
            continue
        token = _extract_token(data)
        if token is None:
            continue
        token_json = _token_to_rclone_json(token)
        # Validate that we actually have an access_token after conversion.
        try:
            parsed = json.loads(token_json)
        except json.JSONDecodeError:
            continue
        if not parsed.get("access_token"):
            continue
        client_id, client_secret = _extract_client_credentials(data)
        # ``account_label`` is not stored by HA; leave it empty and let the
        # backend derive it from the token if needed.
        return {
            "token_json": token_json,
            "client_id": client_id,
            "client_secret": client_secret,
            "account_label": "",
        }
    return None


async def async_push_ha_dropbox_token(hass: HomeAssistant, ws_client: Any) -> bool:
    """Push the HA Dropbox token to the backend if available.

    Returns True when a token was sent, False when no entry/token exists or
    the WebSocket is not connected.
    """
    # ``get_dropbox_token_payload`` is synchronous and only inspects in-memory
    # dicts, so calling it directly is fine and keeps tests simple. Avoid
    # ``async_add_executor_job`` indirection which previously caused a double
    # invocation bug.
    payload = get_dropbox_token_payload(hass)
    if payload is None:
        _LOGGER.debug("No HA Dropbox token available to forward")
        return False

    # Defer to the WebSocket client helper (added in websocket_client.py).
    try:
        await ws_client.async_send_ha_dropbox_token(
            token_json=payload["token_json"],
            client_id=payload.get("client_id", ""),
            client_secret=payload.get("client_secret", ""),
            account_label=payload.get("account_label", ""),
        )
    except Exception:
        _LOGGER.debug("Failed to send ha_dropbox_token", exc_info=True)
        return False
    _LOGGER.info("Forwarded HA Dropbox token to SRAT backend")
    return True
