"""Home Assistant Repairs proxy support for SRAT."""

from __future__ import annotations

from collections.abc import Callable
import logging
from typing import Any

from homeassistant.components.repairs import ConfirmRepairFlow, RepairsFlow
from homeassistant.core import HomeAssistant
from homeassistant.data_entry_flow import FlowResult
from homeassistant.helpers import issue_registry as ir

from .const import DOMAIN
from .websocket_client import SRATWebSocketClient

_LOGGER = logging.getLogger(__name__)


def _severity_from_string(value: str) -> ir.IssueSeverity:
    """Map backend severity strings to Home Assistant issue severities."""
    normalized = value.lower()
    if normalized == "critical":
        return ir.IssueSeverity.CRITICAL
    if normalized == "error":
        return ir.IssueSeverity.ERROR
    return ir.IssueSeverity.WARNING


class SRATRepairProxy:
    """Proxy backend repair commands into Home Assistant issue registry actions."""

    def __init__(self, hass: HomeAssistant, ws_client: SRATWebSocketClient) -> None:
        """Initialize the repair proxy."""
        self._hass = hass
        self._ws_client = ws_client
        self._remove_listener: Callable[[], None] | None = None

    def register(self) -> None:
        """Register websocket listener for repair commands."""
        self._remove_listener = self._ws_client.register_listener(
            "repair_command",
            self._on_repair_command,
        )

    def unregister(self) -> None:
        """Unregister websocket listener for repair commands."""
        if self._remove_listener is not None:
            self._remove_listener()
            self._remove_listener = None

    def _on_repair_command(self, payload: Any) -> None:
        """Handle repair command payloads from websocket events."""
        self._hass.async_create_task(
            self.async_handle_repair_command(payload),
            "srat_repair_command",
        )

    async def async_handle_repair_command(self, payload: Any) -> None:
        """Translate backend repair commands to HA issue operations."""
        if not isinstance(payload, dict):
            _LOGGER.warning("Invalid repair command payload type: %s", type(payload))
            return

        repair_id = payload.get("repair_id")
        action = str(payload.get("action", "")).lower()
        command_id = payload.get("command_id")
        if not isinstance(repair_id, str) or not repair_id:
            _LOGGER.warning("Invalid repair_id in payload: %s", payload)
            return

        try:
            if action in {"upsert", "reconcile"}:
                translation_key = payload.get("translation_key")
                if not isinstance(translation_key, str) or not translation_key:
                    raise ValueError("translation_key is required for upsert/reconcile")

                translation_placeholders = payload.get("translation_placeholders")
                if not isinstance(translation_placeholders, dict):
                    translation_placeholders = None

                data = payload.get("data")
                if not isinstance(data, dict):
                    data = None

                breaks_in_ha_version = payload.get("breaks_in_ha_version")
                if not isinstance(breaks_in_ha_version, str):
                    breaks_in_ha_version = None

                learn_more_url = payload.get("learn_more_url")
                if not isinstance(learn_more_url, str):
                    learn_more_url = None

                is_fixable = bool(payload.get("is_fixable", False))
                is_persistent = bool(payload.get("is_persistent", False))
                severity = _severity_from_string(
                    str(payload.get("severity", "warning"))
                )

                ir.async_create_issue(
                    self._hass,
                    DOMAIN,
                    repair_id,
                    breaks_in_ha_version=breaks_in_ha_version,
                    data=data,
                    is_fixable=is_fixable,
                    is_persistent=is_persistent,
                    learn_more_url=learn_more_url,
                    severity=severity,
                    translation_key=translation_key,
                    translation_placeholders=translation_placeholders,
                )

                await self._ws_client.async_send_repair_lifecycle_event(
                    repair_id=repair_id,
                    command_id=command_id if isinstance(command_id, str) else None,
                    status="created" if action == "upsert" else "updated",
                )
                return

            if action == "delete":
                ir.async_delete_issue(self._hass, DOMAIN, repair_id)
                await self._ws_client.async_send_repair_lifecycle_event(
                    repair_id=repair_id,
                    command_id=command_id if isinstance(command_id, str) else None,
                    status="deleted",
                )
                return

            raise ValueError(f"unsupported repair action: {action}")
        except Exception as err:
            _LOGGER.exception("Failed to handle repair command %s", repair_id)
            await self._ws_client.async_send_repair_lifecycle_event(
                repair_id=repair_id,
                command_id=command_id if isinstance(command_id, str) else None,
                status="error",
                error=str(err),
            )


class SRATIssueRepairFlow(ConfirmRepairFlow):
    """Repair flow that reports successful fixes back to the SRAT backend."""

    def __init__(self, issue_id: str, ws_client: SRATWebSocketClient | None) -> None:
        """Initialize the flow with issue and websocket client references."""
        super().__init__()
        self.issue_id = issue_id
        self._ws_client = ws_client

    async def async_step_confirm(
        self, user_input: dict[str, str] | None = None
    ) -> FlowResult:
        """Handle confirmation and report successful fix events."""
        result = await super().async_step_confirm(user_input)
        if user_input is not None and self._ws_client is not None:
            await self._ws_client.async_send_repair_lifecycle_event(
                repair_id=self.issue_id,
                status="fixed",
            )
        return result


def _resolve_ws_client(hass: HomeAssistant) -> SRATWebSocketClient | None:
    """Resolve active SRAT websocket client from loaded config entries."""
    for entry in hass.config_entries.async_entries(DOMAIN):
        runtime_data = getattr(entry, "runtime_data", None)
        ws_client = getattr(runtime_data, "ws_client", None)
        if isinstance(ws_client, SRATWebSocketClient):
            return ws_client
    return None


async def async_create_fix_flow(
    hass: HomeAssistant,
    issue_id: str,
    data: dict[str, str | int | float | None] | None,
) -> RepairsFlow:
    """Create a Repairs flow for SRAT-managed fixable issues."""
    del data
    return SRATIssueRepairFlow(issue_id, _resolve_ws_client(hass))
