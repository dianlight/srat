"""Constants for the SRAT Companion integration."""
from __future__ import annotations

from typing import Final

DOMAIN: Final = "srat_companion"
MANUFACTURER: Final = "SRAT"
MODEL: Final = "SRAT Companion"

# Configuration
CONF_ADDON_SLUGS: Final = "addon_slugs"
CONF_MANUAL_IP: Final = "manual_ip"
CONF_MANUAL_PORT: Final = "manual_port"
CONF_DISCOVERY_METHOD: Final = "discovery_method"

# Discovery methods
DISCOVERY_ZEROCONF: Final = "zeroconf"
DISCOVERY_ADDON: Final = "addon"
DISCOVERY_MANUAL: Final = "manual"

# Default values
DEFAULT_ADDON_SLUGS: Final = ["local_sambanas2", "c9a35110_sambanas2", "1a32f091_sambanas2"]
DEFAULT_PORT: Final = 3000
DEFAULT_TIMEOUT: Final = 30

# SSE endpoint
SSE_ENDPOINT: Final = "/api/events"
ZEROCONF_SERVICE: Final = "_srat._tcp.local."

# Error codes
ERROR_CANNOT_CONNECT: Final = "cannot_connect"
ERROR_INVALID_AUTH: Final = "invalid_auth"
ERROR_UNKNOWN: Final = "unknown"
