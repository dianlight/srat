"""Config flow for SRAT Companion integration."""

from __future__ import annotations

import asyncio
import logging
from typing import Any

import aiohttp
import voluptuous as vol
from homeassistant import config_entries
from homeassistant.components import zeroconf
from homeassistant.const import CONF_HOST, CONF_PORT
from homeassistant.core import HomeAssistant
from homeassistant.data_entry_flow import FlowResult
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.selector import (
    SelectSelector,
    SelectSelectorConfig,
    SelectSelectorMode,
)
from zeroconf import ServiceBrowser, ServiceListener, Zeroconf
from zeroconf.asyncio import AsyncZeroconf

from .const import (
    CONF_DISCOVERY_METHOD,
    CONF_MANUAL_IP,
    CONF_MANUAL_PORT,
    DEFAULT_ADDON_SLUGS,
    DEFAULT_PORT,
    DISCOVERY_ADDON,
    DISCOVERY_MANUAL,
    DISCOVERY_ZEROCONF,
    DOMAIN,
    ERROR_CANNOT_CONNECT,
    ZEROCONF_SERVICE,
)

_LOGGER = logging.getLogger(__name__)


class SratServiceListener(ServiceListener):
    """Zeroconf service listener for SRAT services."""

    def __init__(self) -> None:
        """Initialize the listener."""
        self.services: dict[str, dict[str, Any]] = {}

    def remove_service(self, zc: Zeroconf, type_: str, name: str) -> None:
        """Remove a service."""
        if name in self.services:
            del self.services[name]

    def add_service(self, zc: Zeroconf, type_: str, name: str) -> None:
        """Add a service."""
        info = zc.get_service_info(type_, name)
        if info:
            self.services[name] = {
                "host": info.parsed_addresses()[0] if info.parsed_addresses() else None,
                "port": info.port,
                "properties": info.properties,
            }

    def update_service(self, zc: Zeroconf, type_: str, name: str) -> None:
        """Update a service."""
        self.add_service(zc, type_, name)


async def validate_connection(
    hass: HomeAssistant, host: str, port: int
) -> dict[str, Any]:
    """Validate that we can connect to the SRAT service."""
    session = async_get_clientsession(hass)

    try:
        async with session.get(
            f"http://{host}:{port}/api/health",
            timeout=aiohttp.ClientTimeout(total=10),
        ) as response:
            if response.status == 200:
                return {"title": f"SRAT Companion at {host}:{port}"}
            return {"error": ERROR_CANNOT_CONNECT}
    except Exception:
        return {"error": ERROR_CANNOT_CONNECT}


class ConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Handle a config flow for SRAT Companion."""

    VERSION = 1

    def __init__(self) -> None:
        """Initialize the config flow."""
        self.discovered_services: dict[str, dict[str, Any]] = {}
        self.addon_services: dict[str, dict[str, Any]] = {}

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> FlowResult:
        """Handle the initial step."""
        if user_input is None:
            return self.async_show_form(
                step_id="user",
                data_schema=vol.Schema(
                    {
                        vol.Required(CONF_DISCOVERY_METHOD): SelectSelector(
                            SelectSelectorConfig(
                                options=[
                                    {
                                        "value": DISCOVERY_ZEROCONF,
                                        "label": "Zeroconf Discovery",
                                    },
                                    {
                                        "value": DISCOVERY_ADDON,
                                        "label": "Addon Discovery",
                                    },
                                    {
                                        "value": DISCOVERY_MANUAL,
                                        "label": "Manual Configuration",
                                    },
                                ],
                                mode=SelectSelectorMode.DROPDOWN,
                            )
                        )
                    }
                ),
            )

        discovery_method = user_input[CONF_DISCOVERY_METHOD]

        if discovery_method == DISCOVERY_ZEROCONF:
            return await self.async_step_zeroconf_discovery()
        if discovery_method == DISCOVERY_ADDON:
            return await self.async_step_addon_discovery()
        return await self.async_step_manual()

    async def async_step_zeroconf_discovery(
        self, user_input: dict[str, Any] | None = None
    ) -> FlowResult:
        """Handle zeroconf discovery."""
        if user_input is None:
            # Discover services
            azc = AsyncZeroconf()
            listener = SratServiceListener()

            try:
                browser = ServiceBrowser(azc.zeroconf, ZEROCONF_SERVICE, listener)
                await asyncio.sleep(3)  # Wait for discovery
                browser.cancel()

                self.discovered_services = listener.services

                if not self.discovered_services:
                    return self.async_show_form(
                        step_id="zeroconf_discovery",
                        errors={"base": "no_services_found"},
                    )

                # Create options for discovered services
                options = []
                for name, info in self.discovered_services.items():
                    if info["host"] and info["port"]:
                        options.append(
                            {
                                "value": name,
                                "label": f"{name} ({info['host']}:{info['port']})",
                            }
                        )

                return self.async_show_form(
                    step_id="zeroconf_discovery",
                    data_schema=vol.Schema(
                        {
                            vol.Required("service"): SelectSelector(
                                SelectSelectorConfig(
                                    options=options,
                                    mode=SelectSelectorMode.DROPDOWN,
                                )
                            )
                        }
                    ),
                )
            finally:
                await azc.async_close()

        # Validate selected service
        selected_service = user_input["service"]
        service_info = self.discovered_services[selected_service]

        result = await validate_connection(
            self.hass, service_info["host"], service_info["port"]
        )

        if "error" in result:
            return self.async_show_form(
                step_id="zeroconf_discovery",
                errors={"base": result["error"]},
            )

        return self.async_create_entry(
            title=result["title"],
            data={
                CONF_HOST: service_info["host"],
                CONF_PORT: service_info["port"],
                CONF_DISCOVERY_METHOD: DISCOVERY_ZEROCONF,
            },
        )

    async def async_step_addon_discovery(
        self, user_input: dict[str, Any] | None = None
    ) -> FlowResult:
        """Handle addon discovery."""
        # Automatically search through the predefined addon slugs
        addon_slugs = DEFAULT_ADDON_SLUGS

        try:
            # Search for running addons
            discovered_addons = await self._discover_addons(addon_slugs)

            if not discovered_addons:
                return self.async_show_form(
                    step_id="addon_discovery",
                    errors={"base": "no_addons_found"},
                    description_placeholders={"addon_slugs": ", ".join(addon_slugs)},
                )

            if user_input is None:
                # Show discovered addons for selection
                options = []
                for slug, info in discovered_addons.items():
                    options.append(
                        {
                            "value": slug,
                            "label": f"{info['name']} ({info['host']}:{info['port']})",
                        }
                    )

                return self.async_show_form(
                    step_id="addon_discovery",
                    data_schema=vol.Schema(
                        {
                            vol.Required("addon"): SelectSelector(
                                SelectSelectorConfig(
                                    options=options,
                                    mode=SelectSelectorMode.DROPDOWN,
                                )
                            )
                        }
                    ),
                )

            # Validate selected addon
            selected_addon = user_input["addon"]
            addon_info = discovered_addons[selected_addon]

            result = await validate_connection(
                self.hass, addon_info["host"], addon_info["port"]
            )

            if "error" in result:
                return self.async_show_form(
                    step_id="addon_discovery",
                    errors={"base": result["error"]},
                )

            return self.async_create_entry(
                title=f"SRAT Companion ({addon_info['name']})",
                data={
                    CONF_HOST: addon_info["host"],
                    CONF_PORT: addon_info["port"],
                    CONF_DISCOVERY_METHOD: DISCOVERY_ADDON,
                    "addon_slug": selected_addon,
                    "addon_name": addon_info["name"],
                },
            )

        except Exception as err:
            _LOGGER.error("Error during addon discovery: %s", err)
            return self.async_show_form(
                step_id="addon_discovery",
                errors={"base": "addon_discovery_failed"},
                description_placeholders={"error": str(err)},
            )

    async def _discover_addons(
        self, addon_slugs: list[str]
    ) -> dict[str, dict[str, Any]]:
        """Discover running addons by slug."""
        discovered = {}

        # This would need to be implemented using Home Assistant Supervisor API
        # For now, we'll simulate the discovery process

        try:
            # Check if we have access to supervisor
            if (
                not hasattr(self.hass, "components")
                or "hassio" not in self.hass.components
            ):
                _LOGGER.debug("Supervisor not available, skipping addon discovery")
                return discovered

            # Import supervisor API (if available)
            try:
                from homeassistant.components.hassio import get_supervisor_client

                supervisor = get_supervisor_client(self.hass)

                # Get list of all addons
                addons_response = await supervisor.addons.list()

                if not addons_response.ok:
                    _LOGGER.error("Failed to get addon list from supervisor")
                    return discovered

                addons_data = addons_response.json()

                # Search through our target addon slugs
                for addon_slug in addon_slugs:
                    for addon in addons_data.get("data", {}).get("addons", []):
                        if (
                            addon.get("slug") == addon_slug
                            and addon.get("state") == "started"
                        ):
                            # Get detailed addon info
                            addon_info_response = await supervisor.addons.info(
                                addon_slug
                            )
                            if addon_info_response.ok:
                                addon_info = addon_info_response.json().get("data", {})

                                # Extract network information
                                network_info = addon_info.get("network", {})
                                host = addon_info.get("ip_address", "127.0.0.1")

                                # Try to find the port from network configuration
                                port = DEFAULT_PORT
                                if network_info:
                                    # Look for HTTP port in network config
                                    for port_key, port_config in network_info.items():
                                        if (
                                            isinstance(port_config, dict)
                                            and "host" in port_config
                                        ):
                                            port = port_config["host"]
                                            break
                                        if isinstance(port_config, int):
                                            port = port_config
                                            break

                                discovered[addon_slug] = {
                                    "host": host,
                                    "port": port,
                                    "name": addon_info.get("name", addon_slug),
                                    "version": addon_info.get("version"),
                                    "slug": addon_slug,
                                }

                                _LOGGER.debug(
                                    "Discovered addon %s at %s:%s",
                                    addon_slug,
                                    host,
                                    port,
                                )
                                break

            except ImportError:
                _LOGGER.debug("Supervisor API not available")

        except Exception as err:
            _LOGGER.error("Error discovering addons: %s", err)
            raise

        return discovered

    async def async_step_manual(
        self, user_input: dict[str, Any] | None = None
    ) -> FlowResult:
        """Handle manual configuration."""
        errors = {}

        if user_input is not None:
            host = user_input[CONF_MANUAL_IP]
            port = user_input[CONF_MANUAL_PORT]

            result = await validate_connection(self.hass, host, port)

            if "error" not in result:
                return self.async_create_entry(
                    title=result["title"],
                    data={
                        CONF_HOST: host,
                        CONF_PORT: port,
                        CONF_DISCOVERY_METHOD: DISCOVERY_MANUAL,
                    },
                )

            errors["base"] = result["error"]

        return self.async_show_form(
            step_id="manual",
            data_schema=vol.Schema(
                {
                    vol.Required(CONF_MANUAL_IP): str,
                    vol.Required(CONF_MANUAL_PORT, default=DEFAULT_PORT): int,
                }
            ),
            errors=errors,
        )

    async def async_step_zeroconf(
        self, discovery_info: zeroconf.ZeroconfServiceInfo
    ) -> FlowResult:
        """Handle zeroconf discovery."""
        if discovery_info.type != ZEROCONF_SERVICE:
            return self.async_abort(reason="not_srat_service")

        host = discovery_info.host
        port = discovery_info.port

        # Check if already configured
        await self.async_set_unique_id(f"{host}:{port}")
        self._abort_if_unique_id_configured()

        # Validate connection
        result = await validate_connection(self.hass, host, port)

        if "error" in result:
            return self.async_abort(reason=result["error"])

        return self.async_create_entry(
            title=result["title"],
            data={
                CONF_HOST: host,
                CONF_PORT: port,
                CONF_DISCOVERY_METHOD: DISCOVERY_ZEROCONF,
            },
        )
