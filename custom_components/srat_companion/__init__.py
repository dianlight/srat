"""The SRAT Companion integration."""
from __future__ import annotations

import asyncio
import json

import logging
from datetime import timedelta
from typing import Any

from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant, callback
from homeassistant.helpers.typing import ConfigType
from homeassistant.exceptions import ConfigEntryNotReady
from homeassistant.loader import Integration
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed
from homeassistant.helpers.issue_registry import IssueRegistry, IssueSeverity, IssueCategory, create_issue, delete_issue, async_get_issue_registry
from homeassistant.helpers.aiohttp_client import async_get_clientsession
import aiohttp
from homeassistant.util import utcnow

_LOGGER = logging.getLogger(__name__)

DOMAIN = "srat_companion"
PLATFORMS: list[str] = [] # This component does not create entities, only discovery and repair.

# --- CONFIGURE THIS SUFFIX FOR THE SLUGS OF THE ADD-ONS YOU WANT TO MONITOR ---
ADDON_SLUG_SUFFIX = "_sambanas2"
# -----------------------------------------------------------------------

# The event name for Supervisor add-on information responses
EVENT_SUPERVISOR_ADDON_INFO = "hassio_addon_info"

SCAN_INTERVAL = timedelta(minutes=1) # How often to check the status of add-ons

# This set will store all add-on slugs that match the pattern and have been "discovered"
# to avoid triggering the discovery flow multiple times.
_DISCOVERED_ADDONS: set[str] = set()

# This dictionary tracks timestamps for add-ons that are not in the "started" state
# Key: addon_slug, Value: datetime of when it was first noticed as not started
_NOT_STARTED_TIMESTAMPS: dict[str, Any] = {}

# This dictionary will track active SSE listener tasks for each add-on
_SSE_CONNECTIONS: dict[str, asyncio.Task[None]] = {}


async def async_setup(hass: HomeAssistant, config: ConfigType) -> bool:
    """Set up SRAT Companion from configuration.yaml (or for initial auto-discovery)."""
    hass.data.setdefault(DOMAIN, {})

    # Check if a config entry for the main integration is already set up.
    existing_entries = hass.config_entries.async_entries(DOMAIN)
    is_already_configured = any(entry.unique_id == DOMAIN for entry in existing_entries)

    if not is_already_configured:
        _LOGGER.debug("Checking for initial auto-discovery of add-ons with suffix: %s", ADDON_SLUG_SUFFIX)
        try:
            hassio_integration: Integration | None = hass.data.get("hassio")
            if not hassio_integration:
                _LOGGER.warning(
                    "Home Assistant Supervisor (hassio) integration was not loaded during startup. "
                    "Auto-discovery might be delayed or fail."
                )
                return True

            # Call the hassio service to get information about ALL add-ons
            response: dict[str, Any] = await hass.services.async_call(
                "hassio", "addons_info", blocking=True, return_response=True
            )
            
            addons_info: dict[str, Any] = response.get("info", {})
            
            for addon_slug, addon_data in addons_info.items():
                if addon_slug.endswith(ADDON_SLUG_SUFFIX):
                    if addon_data.get("installed", False) and addon_slug not in _DISCOVERED_ADDONS:
                        _LOGGER.info(
                            "Add-on '%s' (ending with '%s') found installed at startup. Triggering auto-discovery.",
                            addon_slug, ADDON_SLUG_SUFFIX
                        )
                        hass.async_create_task(
                            hass.config_entries.flow.async_init(
                                DOMAIN,
                                context={"source": hass.config_entries.SOURCE_DISCOVERY},
                                data={"addon": addon_slug},
                            )
                        )
                        _DISCOVERED_ADDONS.add(addon_slug)
        except Exception as err:
            _LOGGER.error(
                "Error during initial auto-discovery for add-ons with suffix '%s': %s",
                ADDON_SLUG_SUFFIX, err
            )
    else:
        _LOGGER.debug("SRAT Companion is already configured.")

    return True


async def _start_sse_listener(hass: HomeAssistant, addon_slug: str, sse_ingress_path: str):
    """Starts an SSE listener for an add-on's ingress endpoint."""
    full_sse_url = f"{sse_ingress_path}/sse"
    _LOGGER.info(
        "Attempting to connect to SSE endpoint for %s at %s",
        addon_slug, full_sse_url
    )
    session = async_get_clientsession(hass)

    try:
        # Use the ingress path directly; HA's client session will handle the correct base URL.
        async with session.get(full_sse_url, timeout=aiohttp.ClientTimeout(total=30)) as resp:
            if resp.status == 200:
                _LOGGER.info(
                    "Successfully connected to SSE for %s. Status: %s",
                    addon_slug, resp.status
                )
                async for line_bytes in resp.content:
                    # SSE messages are expected to be UTF-8
                    line = line_bytes.decode('utf-8').strip()

                    if not line:  # An empty line signifies the end of an event block
                        if current_event_name and data_buffer:
                            full_data_str = "".join(data_buffer)
                            try:
                                parsed_data = json.loads(full_data_str)
                                _LOGGER.info(
                                    "SSE Event from %s - Type: '%s', Data: %s",
                                    addon_slug, current_event_name, parsed_data
                                )
                                # TODO: Implement specific logic based on event_name and parsed_data
                                # For example:
                                # if current_event_name == "hello":
                                #     _LOGGER.debug("Welcome message received: %s", parsed_data)
                                # elif current_event_name == "heartbeat":
                                #     _LOGGER.debug("Heartbeat received: %s", parsed_data)
                                # elif current_event_name == "volumes":
                                #     _LOGGER.debug("Volumes update: %s", parsed_data)
                                # elif current_event_name == "updating":
                                #     _LOGGER.debug("Update progress: %s", parsed_data)
                                # elif current_event_name == "share":
                                #     _LOGGER.debug("Share update: %s", parsed_data)

                            except json.JSONDecodeError:
                                _LOGGER.error(
                                    "SSE JSONDecodeError for event '%s' from %s. Raw data: '%s'",
                                    current_event_name, addon_slug, full_data_str
                                )
                            except Exception as e_dispatch:
                                _LOGGER.error(
                                    "Exception while dispatching SSE event '%s' from %s: %s",
                                    current_event_name, addon_slug, e_dispatch
                                )
                        # Reset for the next event
                        current_event_name = None # Default event type if not specified
                        data_buffer = []
                        continue

                    if line.startswith(':'):  # Comment line, typically for keep-alive
                        _LOGGER.debug("SSE keep-alive/comment from %s: %s", addon_slug, line)
                        continue

                    # Try to split the line into field and value
                    # SSE lines are typically "field: value"
                    parts = line.split(":", 1)
                    field = parts[0].strip()
                    value = parts[1].strip() if len(parts) > 1 else ""

                    if field == "event":
                        current_event_name = value
                    elif field == "data":
                        data_buffer.append(value) # Data can be split across multiple lines, join later
                    elif field == "id":
                        _LOGGER.debug("SSE event ID from %s: %s", addon_slug, value)
                    elif field == "retry":
                        _LOGGER.debug("SSE retry value from %s: %s ms", addon_slug, value)
                    else:
                        _LOGGER.debug("SSE unknown line from %s: %s", addon_slug, line)
            else:
                _LOGGER.error(
                    "Failed to connect to SSE for %s. Status: %s, Response: %s",
                    addon_slug,
                    resp.status,
                    await resp.text()
                )

        # After the loop, process any lingering event data if the stream ended abruptly
        if current_event_name and data_buffer:
            full_data_str = "".join(data_buffer)
            try:
                parsed_data = json.loads(full_data_str)
                _LOGGER.info(
                    "SSE Event (EOF) from %s - Type: '%s', Data: %s",
                    addon_slug, current_event_name, parsed_data
                )
            except json.JSONDecodeError:
                _LOGGER.error(
                    "SSE JSONDecodeError (EOF) for event '%s' from %s. Raw data: '%s'",
                    current_event_name, addon_slug, full_data_str
                )
            except Exception as e_dispatch:
                _LOGGER.error(
                    "Exception while dispatching SSE event (EOF) '%s' from %s: %s",
                    current_event_name, addon_slug, e_dispatch
                )

    except aiohttp.ClientConnectorError as e:
        _LOGGER.error("SSE connection error for %s (%s): %s", addon_slug, full_sse_url, e)
    except asyncio.TimeoutError:
        _LOGGER.error("SSE connection timeout for %s (%s)", addon_slug, full_sse_url)
    except Exception as e:
        _LOGGER.error("Exception during SSE handling for %s (%s): %s", addon_slug, full_sse_url, e)
    finally:
        _LOGGER.info("SSE listener for %s terminated.", addon_slug)
        _SSE_CONNECTIONS.pop(addon_slug, None) # Remove the task if it finishes on its own

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
            await hass.services.async_call(
                "hassio", "addons_info", blocking=False
            )
            return True
        except Exception as err:
            _LOGGER.error("Error calling hassio.addons_info: %s", err)
            raise UpdateFailed(f"Errore nella chiamata hassio.addons_info: {err}") from err

    coordinator = DataUpdateCoordinator(
        hass,
        _LOGGER,
        name="Supervisor Add-on Status",
        update_method=async_update_data,
        update_interval=SCAN_INTERVAL,
    )
    hass.data[DOMAIN][entry.entry_id] = {"coordinator": coordinator} # Store the coordinator

    @callback
    def _async_hassio_addons_info_listener(event):
        """Listen for hassio_addon_info events (which include info for all add-ons)."""
        all_addons_info: dict[str, Any] = event.data.get("info", {})
        issue_registry: IssueRegistry = async_get_issue_registry(hass)

        for addon_slug, addon_info in all_addons_info.items():
            if addon_slug.endswith(ADDON_SLUG_SUFFIX):
                current_state = addon_info.get("state")
                is_installed = addon_info.get("installed", False)
                ingress_url_path = addon_info.get("ingress_url") # Es. /api/hassio_ingress/XXXXX
                # E.g. /api/hassio_ingress/XXXXX

                # --- Discovery Logic ---
                if is_installed and addon_slug not in _DISCOVERED_ADDONS:
                    _LOGGER.info(
                        "Add-on '%s' (ending with '%s') is installed. Triggering discovery.",
                        addon_slug, ADDON_SLUG_SUFFIX
                    )
                    hass.async_create_task(
                        hass.config_entries.flow.async_init(
                            DOMAIN,
                            context={"source": hass.config_entries.SOURCE_DISCOVERY},
                            data={"addon": addon_slug},
                        )
                    )
                    _DISCOVERED_ADDONS.add(addon_slug)

                # --- SSE Connection Logic ---
                if is_installed and current_state == "started" and ingress_url_path:
                    if addon_slug not in _SSE_CONNECTIONS:
                        _LOGGER.info(
                            "Add-on '%s' is running and supports ingress. Starting SSE listener.",
                            addon_slug
                        )
                        sse_task = hass.async_create_task(
                            _start_sse_listener(hass, addon_slug, ingress_url_path)
                        )
                        _SSE_CONNECTIONS[addon_slug] = sse_task
                elif addon_slug in _SSE_CONNECTIONS: # If not started/installed or no ingress
                    _LOGGER.info(
                        "Add-on '%s' is not running, uninstalled, or no longer has ingress. Stopping SSE listener.",
                        addon_slug
                    )
                    task_to_cancel = _SSE_CONNECTIONS.pop(addon_slug)
                    if task_to_cancel and not task_to_cancel.done():
                        task_to_cancel.cancel()

                # --- Repair Logic ---
                # Repair issue ID specific to this addon
                repair_issue_id = f"{DOMAIN}_not_started_{addon_slug}"

                if is_installed and current_state != "started":
                    if addon_slug not in _NOT_STARTED_TIMESTAMPS:
                        # First time we see it not started, record the timestamp
                        _NOT_STARTED_TIMESTAMPS[addon_slug] = utcnow()
                        _LOGGER.debug(
                            "Add-on '%s' is installed but not started. Tracking time.",
                            addon_slug
                        )
                    elif (utcnow() - _NOT_STARTED_TIMESTAMPS[addon_slug]) > timedelta(minutes=5):
                        # More than 5 minutes, create/update the repair issue
                        _LOGGER.warning(
                            "Add-on '%s' has been installed but not started for more than 5 minutes. Creating repair issue.",
                            addon_slug
                        )
                        create_issue(
                            hass,
                            DOMAIN,
                            repair_issue_id, # Dynamic issue ID
                            issue_domain=DOMAIN,
                            is_fixable=True,
                            is_persistent=True,
                            learn_more_url="https://www.home-assistant.io/integrations/hassio/", # Example URL
                            severity=IssueSeverity.WARNING,
                            translation_key="addon_not_started",
                            translation_placeholders={
                                "addon_slug": addon_slug,
                                "time_threshold": "5 minutes"
                            },
                            fix_flow=f"{DOMAIN}_start_addon_fix", # Refers to the repair flow name in config_flow.py
                            context={"addon_slug": addon_slug} # Pass the slug to the fix_flow context
                        )
                elif current_state == "started":
                    # The add-on is started, clear the timestamp and delete the repair issue if it exists
                    if addon_slug in _NOT_STARTED_TIMESTAMPS:
                        del _NOT_STARTED_TIMESTAMPS[addon_slug]
                        _LOGGER.debug(
                            "Add-on '%s' is now started. Clearing not_started_since timestamp.",
                            addon_slug
                        )
                    if issue_registry.get_issue(DOMAIN, repair_issue_id):
                        _LOGGER.info(
                            "Add-on '%s' is now started. Deleting repair issue.",
                            addon_slug
                        )
                        delete_issue(hass, DOMAIN, repair_issue_id)
                # Se non è installato e non era stato rilevato, assicurati che non ci siano issue residue
                elif not is_installed:
                    if addon_slug in _DISCOVERED_ADDONS:
                        _DISCOVERED_ADDONS.remove(addon_slug)
                        _LOGGER.debug("Add-on '%s' is no longer installed. Removed from discovery cache.", addon_slug)
                    if addon_slug in _NOT_STARTED_TIMESTAMPS:
                        del _NOT_STARTED_TIMESTAMPS[addon_slug]
                        _LOGGER.debug("Add-on '%s' is no longer installed. Removed from timestamp tracking.", addon_slug)
                    if issue_registry.get_issue(DOMAIN, repair_issue_id):
                        delete_issue(hass, DOMAIN, repair_issue_id)
                        _LOGGER.info("Add-on '%s' is no longer installed. Deleting repair issue.", addon_slug)
                    # Also ensure the SSE listener is stopped if the add-on is not installed
                    if addon_slug in _SSE_CONNECTIONS:
                        _LOGGER.info("Add-on '%s' is no longer installed. Stopping SSE listener.", addon_slug)
                        task_to_cancel = _SSE_CONNECTIONS.pop(addon_slug)
                        if task_to_cancel and not task_to_cancel.done():
                            task_to_cancel.cancel()
                        delete_issue(hass, DOMAIN, repair_issue_id)
                        _LOGGER.info("Add-on '%s' is no longer installed. Deleting repair issue.", addon_slug) # This seems redundant


    # Listen for the hassio_addon_info event
    # Note: the hassio.addons_info service publishes a hassio_addon_info event
    # containing information for ALL add-ons.
    entry.async_on_unload(
        hass.bus.async_listen(EVENT_SUPERVISOR_ADDON_INFO, _async_hassio_addons_info_listener)
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
        for addon_slug, task in list(_SSE_CONNECTIONS.items()): # Iterate over a copy
            _LOGGER.info("Stopping SSE listener for %s during integration unload.", addon_slug)
            if not task.done():
                task.cancel()
            # Optionally wait for task cancellation with a timeout
            # try:
            #     await asyncio.wait_for(task, timeout=5.0)
            # except (asyncio.TimeoutError, asyncio.CancelledError):
            #     _LOGGER.debug("Timeout or cancellation while waiting for SSE task for %s.", addon_slug)
        _SSE_CONNECTIONS.clear()

        # Clear the global cache and repair issues for all monitored add-ons
        issue_registry: IssueRegistry = async_get_issue_registry(hass)
        for addon_slug in list(_DISCOVERED_ADDONS): # Iterate over a copy to allow modification
            _DISCOVERED_ADDONS.remove(addon_slug)
            repair_issue_id = f"{DOMAIN}_not_started_{addon_slug}"
            if issue_registry.get_issue(DOMAIN, repair_issue_id):
                delete_issue(hass, DOMAIN, repair_issue_id)
                _LOGGER.info("Deleting repair issue '%s' during integration unload.", repair_issue_id)
        
        _NOT_STARTED_TIMESTAMPS.clear() # Clear all timestamps

    return unload_ok
