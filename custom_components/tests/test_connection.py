"""Tests for SRAT connection helpers."""

from __future__ import annotations

from custom_components.srat.connection import iter_connection_hosts
from custom_components.srat.const import LOOPBACK_HOST, SUPERVISOR_GATEWAY_HOST


def test_supervisor_addon_prefers_loopback_then_gateway() -> None:
    """Supervisor-managed add-ons try loopback before gateway and hostname."""
    assert iter_connection_hosts("local-sambanas2", "local_sambanas2") == (
        LOOPBACK_HOST,
        SUPERVISOR_GATEWAY_HOST,
        "local-sambanas2",
    )


def test_core_hostname_without_slug_prefers_loopback_then_gateway() -> None:
    """core-/local- hostnames get loopback and gateway candidates without a slug."""
    assert iter_connection_hosts("core-sambanas2") == (
        LOOPBACK_HOST,
        SUPERVISOR_GATEWAY_HOST,
        "core-sambanas2",
    )


def test_empty_host_returns_loopback_and_gateway() -> None:
    """Empty hosts fall back to loopback first, then the gateway."""
    assert iter_connection_hosts("  ") == (LOOPBACK_HOST, SUPERVISOR_GATEWAY_HOST)


def test_manual_host_is_unchanged() -> None:
    """Manually configured hosts are used as-is without extra candidates."""
    assert iter_connection_hosts("192.168.1.100") == ("192.168.1.100",)


def test_gateway_host_is_deduplicated() -> None:
    """An explicit gateway host does not duplicate candidates."""
    assert iter_connection_hosts(SUPERVISOR_GATEWAY_HOST, "local_sambanas2") == (
        LOOPBACK_HOST,
        SUPERVISOR_GATEWAY_HOST,
    )
