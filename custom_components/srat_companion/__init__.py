"""The SRAT Companion integration."""

from __future__ import annotations

import asyncio
import json
import logging
from datetime import timedelta
from typing import TYPE_CHECKING, Any

import aiohttp
import homeassistant
import homeassistant.config_entries
import homeassistant.core
from homeassistant.core import HomeAssistant, callback
from homeassistant.exceptions import ConfigEntryNotReady, HomeAssistantError
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.issue_registry import (
    IssueRegistry,
    IssueSeverity,
    async_get,
    create_issue,
    delete_issue,
)
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed
from homeassistant.util.dt import utcnow

if TYPE_CHECKING:
    from homeassistant.config_entries import ConfigEntry
    from homeassistant.helpers.typing import ConfigType
    from homeassistant.loader import Integration

_LOGGER = logging.getLogger(__name__)

DOMAIN = "srat_companion"
PLATFORMS: list[str] = []  # This component manages add-on discovery and repair flows

# --- CONFIGURE THIS SUFFIX FOR THE SLUGS OF THE ADD-ONS YOU WANT TO MONITOR ---
ADDON_SLUG_SUFFIX = "_sambanas2"
# -----------------------------------------------------------------------

# The event name for Supervisor add-on information responses
EVENT_SUPERVISOR_ADDON_INFO = "hassio_addon_info"

SCAN_INTERVAL = timedelta(minutes=1)  # How often to check the status of add-ons

# This set will store all add-on slugs that match the pattern and have been "discovered"
# to avoid triggering the discovery flow multiple times.
_DISCOVERED_ADDONS: set[str] = set()

# This dictionary tracks timestamps for add-ons that are not in the "started" state
# Key: addon_slug, Value: datetime of when it was first noticed as not started
_NOT_STARTED_TIMESTAMPS: dict[str, Any] = {}

# This dictionary will track active SSE listener tasks for each add-on
_SSE_CONNECTIONS: dict[str, asyncio.Task[None]] = {}


async def async_setup(hass: HomeAssistant, _: ConfigType) -> bool:
    """Set up SRAT Companion from configuration.yaml (or for initial auto-discovery)."""
    hass.data.setdefault(DOMAIN, {})

    # Check if a config entry for the main integration is already set up.
    existing_entries = hass.config_entries.async_entries(DOMAIN)
    is_already_configured = any(entry.unique_id == DOMAIN for entry in existing_entries)

    if not is_already_configured:
        _LOGGER.debug(
            "Checking for initial auto-discovery of add-ons with suffix: %s",
            ADDON_SLUG_SUFFIX,
        )
        try:
            hassio_integration: Integration | None = hass.data.get("hassio")
            if not hassio_integration:
                _LOGGER.warning(
                    "Home Assistant Supervisor (hassio) integration was not loaded during startup. "
                    "Auto-discovery might be delayed or fail."
                )
                return True

            # Call the hassio service to get information about ALL add-ons
            response = await hass.services.async_call(
                "hassio", "addons_info", blocking=True, return_response=True
            )

            if not isinstance(response, dict) or "info" not in response:
                _LOGGER.error("Invalid response format from hassio addons_info service")
                return False

            addons_info: Any = response["info"]

            for addon_slug, addon_data in addons_info.items():
                if not isinstance(addon_slug, str):
                    _LOGGER.error("Invalid addon_slug type: %s", type(addon_slug))
                    continue

                if (
                    addon_slug.endswith(ADDON_SLUG_SUFFIX)
                    and addon_data.get("installed", False)
                    and addon_slug not in _DISCOVERED_ADDONS
                ):
                    _LOGGER.info(
                        "Add-on '%s' (ending with '%s') found installed at startup. Triggering auto-discovery.",
                        addon_slug,
                        ADDON_SLUG_SUFFIX,
                    )
                    hass.async_create_task(
                        hass.config_entries.flow.async_init(
                            DOMAIN,
                            context={
                                "source": homeassistant.config_entries.SOURCE_DISCOVERY
                            },
                            data={"addon": addon_slug},
                        )
                    )
                    _DISCOVERED_ADDONS.add(addon_slug)
        except (aiohttp.ClientError, HomeAssistantError):
            _LOGGER.exception(
                "Error during initial auto-discovery for add-ons with suffix '%s'",
                ADDON_SLUG_SUFFIX,
            )
            return False
    else:
        _LOGGER.debug("SRAT Companion is already configured.")

    return True


def _parse_sse_line(line: str) -> tuple[str, str]:
    """Parse a single SSE line into field and value."""
    if line.startswith(":"):
        return "comment", line[1:].strip()
    parts = line.split(":", 1)
    field = parts[0].strip()
    value = parts[1].strip() if len(parts) > 1 else ""
    return field, value


def _dispatch_sse_event(
    addon_slug: str, event_name: str, data_buffer: list[str]
) -> None:
    """Dispatch a complete SSE event."""
    full_data_str = "".join(data_buffer)
    try:
        parsed_data = json.loads(full_data_str)
        _LOGGER.info(
            "SSE Event from %s - Type: '%s', Data: %s",
            addon_slug,
            event_name,
            parsed_data,
        )
        # TODO: Implement specific logic based on event_name and parsed_data
    except json.JSONDecodeError:
        _LOGGER.exception(
            "SSE JSONDecodeError for event '%s' from %s. Raw data: '%s'",
            event_name,
            addon_slug,
            full_data_str,
        )
    except Exception:
        _LOGGER.exception(
            "Exception while dispatching SSE event '%s' from %s",
            event_name,
            addon_slug,
        )


def _handle_sse_line(
    addon_slug: str,
    line: str,
    current_event_name: str | None,
    data_buffer: list[str],
) -> tuple[str | None, list[str]]:
    """Handle a single SSE line and update event state."""
    if not line:
        if current_event_name and data_buffer:
            _dispatch_sse_event(addon_slug, current_event_name, data_buffer)
        return None, []
    field, value = _parse_sse_line(line)
    if field == "comment":
        _LOGGER.debug("SSE keep-alive/comment from %s: %s", addon_slug, value)
    elif field == "event":
        current_event_name = value
    elif field == "data":
        data_buffer.append(value)
    elif field == "id":
        _LOGGER.debug("SSE event ID from %s: %s", addon_slug, value)
    elif field == "retry":
        _LOGGER.debug("SSE retry value from %s: %s ms", addon_slug, value)
    else:
        _LOGGER.debug("SSE unknown line from %s: %s", addon_slug, line)
    return current_event_name, data_buffer


async def _start_sse_listener(
    hass: HomeAssistant, addon_slug: str, sse_ingress_path: str
) -> None:
    """Start an SSE listener for an add-on's ingress endpoint."""
    full_sse_url = f"{sse_ingress_path}/sse"
    _LOGGER.info(
        "Attempting to connect to SSE endpoint for %s at %s", addon_slug, full_sse_url
    )
    session = async_get_clientsession(hass)
    current_event_name = None
    data_buffer = []

    SSE_SUCCESS_STATUS = 200
    try:
        async with session.get(
            full_sse_url, timeout=aiohttp.ClientTimeout(total=30)
        ) as resp:
            if resp.status == SSE_SUCCESS_STATUS:
                _LOGGER.info(
                    "Successfully connected to SSE for %s. Status: %s",
                    addon_slug,
                    resp.status,
                )
                async for line_bytes in resp.content:
                    line = line_bytes.decode("utf-8").strip()
                    current_event_name, data_buffer = _handle_sse_line(
                        addon_slug, line, current_event_name, data_buffer
                    )
            else:
                _LOGGER.error(
                    "Failed to connect to SSE for %s. Status: %s, Response: %s",
                    addon_slug,
                    resp.status,
                    await resp.text(),
                )

        # After the loop, process any lingering event data if the stream ended abruptly
        if current_event_name and data_buffer:
            _dispatch_sse_event(addon_slug, current_event_name, data_buffer)

    except aiohttp.ClientConnectorError as e:
        _LOGGER.exception(
            "SSE connection error for %s (%s): %s", addon_slug, full_sse_url, e
        )
    except TimeoutError:
        _LOGGER.exception(
            "SSE connection timeout for %s (%s)", addon_slug, full_sse_url
        )
    except aiohttp.ClientError:
        _LOGGER.exception(
            "aiohttp.ClientError during SSE handling for %s (%s)",
            addon_slug,
            full_sse_url,
        )
    except asyncio.CancelledError:
        _LOGGER.info("SSE listener for %s was cancelled.", addon_slug)
        raise
    finally:
        _LOGGER.info("SSE listener for %s terminated.", addon_slug)
        _SSE_CONNECTIONS.pop(
            addon_slug, None
        )  # Remove the task if it finishes on its own


async def async_setup_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Set up SRAT Companion from a config entry."""
    _LOGGER.debug("Setting up config entry for SRAT Companion: %s", entry.unique_id)

    hassio_integration: Integration | None = hass.data.get("hassio")
    if not hassio_integration:
        _LOGGER.error(
            "Home Assistant Supervisor (hassio) integration was not loaded. "
            "SRAT Companion requires hassio."
        )
        raise ConfigEntryNotReady("L'integrazione Hass.io non è pronta.")

    # The coordinator will update information for ALL add-ons
    async def async_update_data():
        """Fetch data from the Supervisor for all add-ons."""
        _LOGGER.debug("Checking status for all Supervisor add-ons.")
        try:
            # Call the hassio service to get information about ALL add-ons
            await hass.services.async_call("hassio", "addons_info", blocking=False)
            return True
        except (aiohttp.ClientError, HomeAssistantError) as err:
            _LOGGER.error("Error calling hassio.addons_info: %s", err)
            raise UpdateFailed(
                f"Errore nella chiamata hassio.addons_info: {err}"
            ) from err

    coordinator = DataUpdateCoordinator(
        hass,
        _LOGGER,
        name="Supervisor Add-on Status",
        update_method=async_update_data,
        update_interval=SCAN_INTERVAL,
    )
    hass.data[DOMAIN][entry.entry_id] = {
        "coordinator": coordinator
    }  # Store the coordinator

    @callback
    def _async_hassio_addons_info_listener(event: homeassistant.core.Event) -> None:
        """Listen for hassio_addon_info events (which include info for all add-ons)."""
        all_addons_info: dict[str, Any] = event.data.get("info", {})
        from homeassistant.helpers.issue_registry import async_get

        issue_registry: IssueRegistry = async_get(hass)
        for addon_slug, addon_info in all_addons_info.items():
            if addon_slug.endswith(ADDON_SLUG_SUFFIX):
                _handle_discovery(hass, addon_slug, addon_info)
                _handle_sse(hass, addon_slug, addon_info)
                _handle_repair(hass, addon_slug, addon_info, issue_registry)
                _handle_cleanup(hass, addon_slug, addon_info, issue_registry)

    def _handle_discovery(
        hass: HomeAssistant, addon_slug: str, addon_info: dict[str, Any]
    ) -> None:
        is_installed = addon_info.get("installed", False)
        if is_installed and addon_slug not in _DISCOVERED_ADDONS:
            _LOGGER.info(
                "Add-on '%s' (ending with '%s') is installed. Triggering discovery.",
                addon_slug,
                ADDON_SLUG_SUFFIX,
            )
            hass.async_create_task(
                hass.config_entries.flow.async_init(
                    DOMAIN,
                    context={"source": homeassistant.config_entries.SOURCE_DISCOVERY},
                    data={"addon": addon_slug},
                )
            )
            _DISCOVERED_ADDONS.add(addon_slug)

    def _handle_sse(
        hass: HomeAssistant, addon_slug: str, addon_info: dict[str, Any]
    ) -> None:
        current_state = addon_info.get("state")
        is_installed = addon_info.get("installed", False)
        ingress_url_path = addon_info.get("ingress_url")
        if is_installed and current_state == "started" and ingress_url_path:
            if addon_slug not in _SSE_CONNECTIONS:
                _LOGGER.info(
                    "Add-on '%s' is running and supports ingress. Starting SSE listener.",
                    addon_slug,
                )
                sse_task = hass.async_create_task(
                    _start_sse_listener(hass, addon_slug, ingress_url_path)
                )
                _SSE_CONNECTIONS[addon_slug] = sse_task
        elif addon_slug in _SSE_CONNECTIONS:
            _LOGGER.info(
                "Add-on '%s' is not running, uninstalled, or no longer has ingress. Stopping SSE listener.",
                addon_slug,
            )
            task_to_cancel = _SSE_CONNECTIONS.pop(addon_slug)
            if task_to_cancel and not task_to_cancel.done():
                task_to_cancel.cancel()

    def _handle_repair(
        hass: HomeAssistant,
        addon_slug: str,
        addon_info: dict[str, Any],
        issue_registry: IssueRegistry,
    ) -> None:
        current_state = addon_info.get("state")
        is_installed = addon_info.get("installed", False)
        repair_issue_id = f"{DOMAIN}_not_started_{addon_slug}"
        if is_installed and current_state != "started":
            if addon_slug not in _NOT_STARTED_TIMESTAMPS:
                _NOT_STARTED_TIMESTAMPS[addon_slug] = utcnow()
                _LOGGER.debug(
                    "Add-on '%s' is installed but not started. Tracking time.",
                    addon_slug,
                )
            elif (utcnow() - _NOT_STARTED_TIMESTAMPS[addon_slug]) > timedelta(
                minutes=5
            ):
                _LOGGER.warning(
                    "Add-on '%s' has been installed but not started for more than 5 minutes. Creating repair issue.",
                    addon_slug,
                )
                create_issue(
                    hass,
                    DOMAIN,
                    repair_issue_id,
                    issue_domain=DOMAIN,
                    is_fixable=True,
                    is_persistent=True,
                    learn_more_url="https://www.home-assistant.io/integrations/hassio/",
                    severity=IssueSeverity.WARNING,
                    translation_key="addon_not_started",
                    translation_placeholders={
                        "addon_slug": addon_slug,
                        "time_threshold": "5 minutes",
                    },
                )
        elif current_state == "started":
            if addon_slug in _NOT_STARTED_TIMESTAMPS:
                del _NOT_STARTED_TIMESTAMPS[addon_slug]
                _LOGGER.debug(
                    "Add-on '%s' is now started. Clearing not_started_since timestamp.",
                    addon_slug,
                )
            if issue_registry.async_get_issue(DOMAIN, repair_issue_id):
                _LOGGER.info(
                    "Add-on '%s' is now started. Deleting repair issue.",
                    addon_slug,
                )
                delete_issue(hass, DOMAIN, repair_issue_id)

    def _handle_cleanup(
        hass: HomeAssistant,
        addon_slug: str,
        addon_info: dict[str, Any],
        issue_registry: IssueRegistry,
    ) -> None:
        is_installed = addon_info.get("installed", False)
        repair_issue_id = f"{DOMAIN}_not_started_{addon_slug}"
        if not is_installed:
            if addon_slug in _DISCOVERED_ADDONS:
                _DISCOVERED_ADDONS.remove(addon_slug)
                _LOGGER.debug(
                    "Add-on '%s' is no longer installed. Removed from discovery cache.",
                    addon_slug,
                )
            if addon_slug in _NOT_STARTED_TIMESTAMPS:
                del _NOT_STARTED_TIMESTAMPS[addon_slug]
                _LOGGER.debug(
                    "Add-on '%s' is no longer installed. Removed from timestamp tracking.",
                    addon_slug,
                )
            if issue_registry.async_get_issue(DOMAIN, repair_issue_id):
                delete_issue(hass, DOMAIN, repair_issue_id)
                _LOGGER.info(
                    "Add-on '%s' is no longer installed. Deleting repair issue.",
                    addon_slug,
                )
            if addon_slug in _SSE_CONNECTIONS:
                _LOGGER.info(
                    "Add-on '%s' is no longer installed. Stopping SSE listener.",
                    addon_slug,
                )
                task_to_cancel = _SSE_CONNECTIONS.pop(addon_slug)
                if task_to_cancel and not task_to_cancel.done():
                    task_to_cancel.cancel()
                delete_issue(hass, DOMAIN, repair_issue_id)
                _LOGGER.info(
                    "Add-on '%s' is no longer installed. Deleting repair issue.",
                    addon_slug,
                )

    # Listen for the hassio_addon_info event
    # Note: the hassio.addons_info service publishes a hassio_addon_info event
    # containing information for ALL add-ons.
    entry.async_on_unload(
        hass.bus.async_listen(
            EVENT_SUPERVISOR_ADDON_INFO, _async_hassio_addons_info_listener
        )
    )

    # Immediately trigger the first update to get the initial state of all add-ons
    await coordinator.async_config_entry_first_refresh()

    return True


async def async_unload_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Unload a config entry."""
    _LOGGER.debug("Unloading config entry for SRAT Companion: %s", entry.entry_id)
    if unload_ok := await hass.config_entries.async_unload_platforms(entry, PLATFORMS):
        # Ensure the coordinator is removed from memory
        coordinator = hass.data[DOMAIN][entry.entry_id].pop("coordinator")
        if hasattr(coordinator, "shutdown"):
            coordinator.shutdown()

        # Completely clear domain data for this entry
        del hass.data[DOMAIN][entry.entry_id]

        # Stop and clear all active SSE listeners
        for addon_slug, task in list(_SSE_CONNECTIONS.items()):  # Iterate over a copy
            _LOGGER.info(
                "Stopping SSE listener for %s during integration unload.", addon_slug
            )
            if not task.done():
                task.cancel()
            # Optionally wait for task cancellation with a timeout
            # try:
            #     await asyncio.wait_for(task, timeout=5.0)
            # except (asyncio.TimeoutError, asyncio.CancelledError):
            #     _LOGGER.debug("Timeout or cancellation while waiting for SSE task for %s.", addon_slug)
        _SSE_CONNECTIONS.clear()

        # Clear the global cache and repair issues for all monitored add-ons
        issue_registry: IssueRegistry = async_get(hass)
        for addon_slug in list(
            _DISCOVERED_ADDONS
        ):  # Iterate over a copy to allow modification
            _DISCOVERED_ADDONS.remove(addon_slug)
            repair_issue_id = f"{DOMAIN}_not_started_{addon_slug}"
            if issue_registry.async_get_issue(DOMAIN, repair_issue_id):
                delete_issue(hass, DOMAIN, repair_issue_id)
                _LOGGER.info(
                    "Deleting repair issue '%s' during integration unload.",
                    repair_issue_id,
                )

        _NOT_STARTED_TIMESTAMPS.clear()  # Clear all timestamps

    return unload_ok
