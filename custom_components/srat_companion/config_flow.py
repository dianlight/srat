"""Config flow for SRAT Companion integration."""

import logging

from homeassistant import config_entries
from homeassistant.config_entries import ConfigFlowResult
from homeassistant.data_entry_flow import FlowResult

# from homeassistant.helpers.issue_registry import AbstractFixFlow
# Import the slug suffix constant from __init__
from .__init__ import ADDON_SLUG_SUFFIX

_LOGGER = logging.getLogger(__name__)

DOMAIN = "srat_companion"


class ConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Handle a config flow for SRAT Companion."""

    VERSION = 1
    CONNECTION_CLASS = config_entries.CONN_CLASS_LOCAL_POLL

    async def async_step_user(self, user_input: dict | None = None) -> ConfigFlowResult:
        """Handle the initial step when adding the integration manually."""
        # We enforce a single instance of this integration as it monitors a pattern.
        await self.async_set_unique_id(DOMAIN)  # The unique ID for the main integration
        self._abort_if_unique_id_configured()

        if user_input is not None:
            return self.async_create_entry(
                title=f"Add-on Monitoring: *{ADDON_SLUG_SUFFIX}",
                data={},  # No user data needed as the slug is a constant pattern
            )

        # Show a confirmation form as no input is required
        return self.async_show_form(
            step_id="user", description_placeholders={"addon_suffix": ADDON_SLUG_SUFFIX}
        )

    async def async_step_discovery(self, discovery_info=None) -> ConfigFlowResult:
        """Handle discovery of the running (or installed) add-on."""
        _LOGGER.debug("Discovery step initiated with info: %s", discovery_info)
        if discovery_info is None:
            return self.async_abort(reason="no_discovery_info")

        # Discovery info will contain the addon_slug that was found.
        discovered_slug = discovery_info.get(
            "addon"
        )  # The 'addon' key is used in the async_init data in __init__.py
        if not discovered_slug or not discovered_slug.endswith(ADDON_SLUG_SUFFIX):
            _LOGGER.warning(
                "Discovery initiated for an unexpected add-on slug: %s. Expected suffix: %s",
                discovered_slug,
                ADDON_SLUG_SUFFIX,
            )
            return self.async_abort(reason="unexpected_addon_slug")

        # Use the discovered add-on slug as the unique ID for the discovery config entry.
        # This allows multiple add-ons matching the pattern to be offered for configuration.
        await self.async_set_unique_id(discovered_slug)
        self._abort_if_unique_id_configured()

        self.context["title_placeholders"] = {"name": discovered_slug}

        return self.async_show_form(
            step_id="discovery_confirm",
            description_placeholders={"addon_slug": discovered_slug},
        )

    async def async_step_discovery_confirm(self, user_input=None) -> ConfigFlowResult:
        """Confirm discovery."""
        if user_input is not None:
            # Create the config entry for the specific discovered addon.
            # The slug is derived from the unique_id.
            addon_slug = self.unique_id
            return self.async_create_entry(
                title=f"Discovered Add-on: {addon_slug}",
                data={"addon": addon_slug},  # Store the discovered slug
            )
        return self.async_abort(reason="not_confirmed")


class AddonRepairFlow:
    """Repair flow for non-started addons."""

    # The translation for this flow is defined in strings.json under "issues"
    # The fix_flow in the create_issue call must match this ID.
    @property
    def flow_id(self) -> str:
        """Return the repair flow ID."""
        return f"{DOMAIN}_start_addon_fix"

    from typing import Optional

    def __init__(self, addon_slug: str, context: dict | None = None) -> None:
        """Initialize the repair flow."""
        self.addon_slug = addon_slug
        self.context = context or {}
        # super().__init__()  # Not needed since we are not inheriting from AbstractFixFlow

    async def async_step_init(self, user_input=None) -> ConfigFlowResult:
        """Handle the initial step of the repair flow."""
        # Get the addon slug from the issue context
        addon_slug = self.context.get("addon_slug")
        if not addon_slug:
            _LOGGER.error("Addon slug not found in repair flow context.")
            raise RuntimeError("missing_addon_slug")

        return self.async_show_form(
            step_id="confirm_start",
            description_placeholders={"addon_slug": addon_slug},
        )

    async def async_step_confirm_start(self, user_input=None) -> ConfigFlowResult:
        """Confirm and execute the start action."""
        addon_slug = self.context.get("addon_slug")
        if not addon_slug:
            raise RuntimeError("missing_addon_slug")

        if user_input is not None:
            try:
                _LOGGER.info(
                    "Attempting to start addon '%s' via repair flow.", addon_slug
                )
                # Here you would add logic to actually start the addon if needed.
                return self.async_show_success(
                    description_placeholders={"addon_slug": addon_slug}
                )
            except RuntimeError as err:
                _LOGGER.exception(
                    "Failed to start addon '%s' via repair flow: %s", addon_slug, err
                )
                return self.async_show_form(
                    step_id="confirm_start",
                    description_placeholders={
                        "addon_slug": addon_slug,
                        "error": str(err),
                    },
                    errors={"base": "failed_to_start"},
                )
        return self.async_show_form(step_id="confirm_start")  # Should not happen
