"""SSE frame parser for SRAT WebSocket text frames."""

from __future__ import annotations

import logging
from typing import Any

_LOGGER = logging.getLogger(__name__)


def parse_sse_frame(data: str) -> dict[str, Any] | None:
    """Parse an SSE-style frame into a dict result.

    Accepts fields in any order and joins multi-line ``data:`` lines.
    Returns ``{"id": ..., "event": ..., "data": ...}`` with data as raw string,
    or ``None`` plus a warning when malformed (no silent drop).
    """
    if not isinstance(data, str) or not data.strip():
        _LOGGER.warning("Malformed SSE frame: empty payload")
        return None
    event: str | None = None
    raw_id: str | None = None
    data_lines: list[str] = []
    for raw_line in data.splitlines():
        # Field detection runs on a stripped view; ``data:`` values are read
        # from the raw line so leading extras and trailing whitespace survive
        # (SSE raw-data contract: only one optional space after the colon).
        line = raw_line.strip()
        if not line:
            continue
        if line.startswith("event:"):
            event = line[len("event:") :].strip()
        elif line.startswith("id:"):
            raw_id = line[len("id:") :].strip()
        elif line.startswith("data:"):
            offset = len(raw_line) - len(raw_line.lstrip())
            value = raw_line[offset + len("data:") :]
            if value.startswith(" "):
                value = value[1:]
            data_lines.append(value)
        else:
            continue
    if not event:
        _LOGGER.warning("Malformed SSE frame: missing event in %r", data[:100])
        return None
    if not data_lines:
        _LOGGER.warning("Malformed SSE frame: missing data in %r", data[:100])
        return None
    return {"id": raw_id, "event": event, "data": "\n".join(data_lines)}
