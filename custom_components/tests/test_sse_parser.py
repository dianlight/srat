"""Tests for SRAT SSE frame parser."""

from __future__ import annotations

from custom_components.srat.sse_parser import parse_sse_frame


def test_parse_in_order() -> None:
    """Parse a standard in-order frame."""
    frame = parse_sse_frame('id: 1\nevent: volumes\ndata: {"a": 1}')
    assert frame == {"id": "1", "event": "volumes", "data": '{"a": 1}'}


def test_parse_out_of_order() -> None:
    """Fields in any order are accepted."""
    frame = parse_sse_frame('data: {"a": 1}\nevent: heartbeat\nid: 2')
    assert frame is not None
    assert frame["event"] == "heartbeat"
    assert frame["id"] == "2"


def test_parse_multiline_data_joined() -> None:
    """Multiple data lines are joined with newlines."""
    frame = parse_sse_frame("event: volumes\ndata: line1\ndata: line2\nid: 3")
    assert frame is not None
    assert frame["data"] == "line1\nline2"


def test_parse_malformed_returns_none() -> None:
    """Missing event or data returns None with warning."""
    assert parse_sse_frame("id: 1\ndata: {}") is None
    assert parse_sse_frame("id: 1\nevent: volumes") is None
    assert parse_sse_frame("") is None
