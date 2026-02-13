"""Config flow for SRAT integration."""

from __future__ import annotations

import asyncio
import logging
from typing import Any

import aiohttp
try:
    from homeassistant.components.hassio.discovery import HassioServiceInfo
except ImportError:  # pragma: no cover - fallback for older HA versions
    from homeassistant.components.hassio import HassioServiceInfo
from homeassistant.config_entries import ConfigFlow, ConfigFlowResult
from homeassistant.helpers.aiohttp_client import async_get_clientsession
import voluptuous as vol

from .const import (
    ADDON_SLUG_WHITELIST,
    CONF_ADDON_SLUG,
    CONF_HOST,
    CONF_PORT,
    DEFAULT_HOST,
    DEFAULT_PORT,
    DOMAIN,
)

_LOGGER = logging.getLogger(__name__)

USER_DATA_SCHEMA = vol.Schema(
    {
        vol.Required(CONF_HOST, default=DEFAULT_HOST): str,
        vol.Required(CONF_PORT, default=DEFAULT_PORT): int,
    }
)


class SRATConfigFlow(ConfigFlow, domain=DOMAIN):  # type: ignore[call-arg]
    """Handle a config flow for SRAT."""

    VERSION = 1

    def __init__(self) -> None:
        """Initialize the config flow."""
        self._discovery_info: dict[str, Any] = {}

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> ConfigFlowResult:
        """Handle the initial step (manual configuration)."""
        errors: dict[str, str] = {}

        if user_input is not None:
            host = user_input[CONF_HOST]
            port = user_input[CONF_PORT]

            error = await self._test_connection(host, port)
            if error:
                errors["base"] = error
            else:
                await self.async_set_unique_id(f"srat_{host}_{port}")
                self._abort_if_unique_id_configured()

                return self.async_create_entry(
                    title=f"SRAT ({host}:{port})",
                    data={CONF_HOST: host, CONF_PORT: port},
                )

        return self.async_show_form(
            step_id="user",
            data_schema=USER_DATA_SCHEMA,
            errors=errors,
        )

    async def async_step_hassio(
        self, discovery_info: HassioServiceInfo
    ) -> ConfigFlowResult:
        """Handle Supervisor add-on discovery."""
        _LOGGER.debug("SRAT hassio discovery: %s", discovery_info)

        slug = discovery_info.slug
        if not slug or slug not in ADDON_SLUG_WHITELIST:
            return self.async_abort(reason="not_srat_addon")

        config = discovery_info.config or {}
        host = config.get("host", DEFAULT_HOST)
        port = config.get("port", DEFAULT_PORT)

        await self.async_set_unique_id(f"srat_{slug or host}")
        self._abort_if_unique_id_configured(updates={CONF_HOST: host, CONF_PORT: port})

        self._discovery_info = {
            CONF_HOST: host,
            CONF_PORT: port,
            CONF_ADDON_SLUG: slug,
        }

        return await self.async_step_hassio_confirm()

    async def async_step_hassio_confirm(
        self, user_input: dict[str, Any] | None = None
    ) -> ConfigFlowResult:
        """Confirm the Supervisor add-on discovery."""
        if user_input is not None:
            return self.async_create_entry(
                title="SRAT",
                data=self._discovery_info,
            )

        return self.async_show_form(
            step_id="hassio_confirm",
            description_placeholders={
                "addon": self._discovery_info.get(CONF_ADDON_SLUG, "SRAT"),
                "host": str(self._discovery_info.get(CONF_HOST, DEFAULT_HOST)),
                "port": str(self._discovery_info.get(CONF_PORT, DEFAULT_PORT)),
            },
        )

    async def _test_connection(self, host: str, port: int) -> str | None:
        """Test connectivity to the SRAT API.

        Returns an error string key if connection fails, None on success.
        """
        session = async_get_clientsession(self.hass)
        try:
            async with asyncio.timeout(10):
                async with session.get(f"http://{host}:{port}/health") as resp:
                    if resp.status != 200:
                        return "cannot_connect"
        except (aiohttp.ClientError, TimeoutError):
            return "cannot_connect"
        except Exception:
            _LOGGER.exception("Unexpected error connecting to SRAT")
            return "unknown"

        return None
