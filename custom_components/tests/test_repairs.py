"""Tests for SRAT repairs proxy behavior."""

from __future__ import annotations

import json
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

from homeassistant.core import HomeAssistant

from custom_components.srat.repairs import SRATRepairProxy
from custom_components.srat.websocket_client import SRATWebSocketClient


async def test_repair_proxy_handles_upsert_command(hass: HomeAssistant) -> None:
    """Test upsert command creates/updates issue and reports lifecycle created."""
    ws_client = AsyncMock(spec=SRATWebSocketClient)
    proxy = SRATRepairProxy(hass=hass, ws_client=ws_client)

    payload = {
        "command_id": "cmd-1",
        "repair_id": "disk_space_low",
        "action": "upsert",
        "translation_key": "disk_space_low",
        "severity": "warning",
        "is_fixable": True,
        "is_persistent": True,
    }

    with patch("custom_components.srat.repairs.ir.async_create_issue") as create_issue:
        await proxy.async_handle_repair_command(payload)

    create_issue.assert_called_once()
    ws_client.async_send_repair_lifecycle_event.assert_awaited_once_with(
        repair_id="disk_space_low",
        command_id="cmd-1",
        status="created",
    )


async def test_repair_proxy_handles_delete_command(hass: HomeAssistant) -> None:
    """Test delete command removes issue and reports lifecycle deleted."""
    ws_client = AsyncMock(spec=SRATWebSocketClient)
    proxy = SRATRepairProxy(hass=hass, ws_client=ws_client)

    payload = {
        "command_id": "cmd-2",
        "repair_id": "disk_space_low",
        "action": "delete",
    }

    with patch("custom_components.srat.repairs.ir.async_delete_issue") as delete_issue:
        await proxy.async_handle_repair_command(payload)

    delete_issue.assert_called_once_with(hass, "srat", "disk_space_low")
    ws_client.async_send_repair_lifecycle_event.assert_awaited_once_with(
        repair_id="disk_space_low",
        command_id="cmd-2",
        status="deleted",
    )


async def test_repair_proxy_reports_error_for_invalid_action(
    hass: HomeAssistant,
) -> None:
    """Test invalid action emits lifecycle error response."""
    ws_client = AsyncMock(spec=SRATWebSocketClient)
    proxy = SRATRepairProxy(hass=hass, ws_client=ws_client)

    payload = {
        "command_id": "cmd-3",
        "repair_id": "disk_space_low",
        "action": "unknown",
    }

    await proxy.async_handle_repair_command(payload)

    ws_client.async_send_repair_lifecycle_event.assert_awaited_once()
    kwargs = ws_client.async_send_repair_lifecycle_event.await_args.kwargs
    assert kwargs["repair_id"] == "disk_space_low"
    assert kwargs["status"] == "error"


def test_repair_proxy_register_and_unregister(hass: HomeAssistant) -> None:
    """Test listener registration lifecycle on websocket client."""
    ws_client = MagicMock(spec=SRATWebSocketClient)
    remove_listener = MagicMock()
    ws_client.register_listener.return_value = remove_listener

    proxy = SRATRepairProxy(hass=hass, ws_client=ws_client)
    proxy.register()
    ws_client.register_listener.assert_called_once()

    proxy.unregister()
    remove_listener.assert_called_once()


# --- Translation key coverage tests ---

_STRINGS_PATH = Path(__file__).parent.parent / "srat" / "strings.json"
_EN_TRANSLATION_PATH = (
    Path(__file__).parent.parent / "srat" / "translations" / "en.json"
)

# All repair translation keys broadcast by the backend that the custom component must define.
_REQUIRED_ISSUE_KEYS = frozenset(
    {
        "custom_component_restart_required",
        "custom_component_missing",
        "addon_config_changed",
    }
)

# Repair keys that are fixable and therefore require a fix_flow.confirm step in translations.
_FIXABLE_ISSUE_KEYS = frozenset(
    {
        "custom_component_restart_required",
        "custom_component_missing",
    }
)


def test_strings_json_defines_all_required_issue_keys() -> None:
    """strings.json must define every repair translation key sent by the backend."""
    data = json.loads(_STRINGS_PATH.read_text(encoding="utf-8"))
    issues = data.get("issues", {})
    missing = _REQUIRED_ISSUE_KEYS - issues.keys()
    assert not missing, f"strings.json is missing issue translation keys: {missing}"


def test_strings_json_fixable_issues_have_fix_flow() -> None:
    """Fixable repair keys in strings.json must declare a fix_flow.step.confirm section."""
    data = json.loads(_STRINGS_PATH.read_text(encoding="utf-8"))
    issues = data.get("issues", {})
    for key in _FIXABLE_ISSUE_KEYS:
        assert key in issues, f"strings.json missing issue key: {key}"
        fix_flow = issues[key].get("fix_flow", {})
        confirm = fix_flow.get("step", {}).get("confirm", {})
        assert confirm.get("title"), (
            f"strings.json issue '{key}' fix_flow.step.confirm.title is missing"
        )
        assert confirm.get("description"), (
            f"strings.json issue '{key}' fix_flow.step.confirm.description is missing"
        )


def test_en_translation_matches_strings_json_issue_keys() -> None:
    """translations/en.json must define the same issue keys as strings.json."""
    strings = json.loads(_STRINGS_PATH.read_text(encoding="utf-8"))
    en = json.loads(_EN_TRANSLATION_PATH.read_text(encoding="utf-8"))
    strings_keys = set(strings.get("issues", {}).keys())
    en_keys = set(en.get("issues", {}).keys())
    assert strings_keys == en_keys, (
        f"Issue key mismatch between strings.json and en.json: "
        f"only in strings.json={strings_keys - en_keys}, only in en.json={en_keys - strings_keys}"
    )
