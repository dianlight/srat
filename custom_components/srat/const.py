"""Constants for the SRAT integration."""

DOMAIN = "srat"

CONF_HOST = "host"
CONF_PORT = "port"
CONF_ADDON_SLUG = "addon_slug"

DEFAULT_PORT = 8099
DEFAULT_HOST = "localhost"

# Addon slugs that can be auto-discovered via the Supervisor API.
# HA slug format: first 8 chars of SHA1(repo_url) + "_" + addon_name.
#   stable:  SHA1("https://github.com/dianlight/hassio-addons")[:8]      = 1a32f091
#   beta:    SHA1("https://github.com/dianlight/hassio-addons-beta")[:8]  = c9a35110
#   local:   local development addons use "local_" prefix
ADDON_SLUG_WHITELIST = [
    "1a32f091_sambanas2",  # stable – dianlight/hassio-addons
    "c9a35110_sambanas2",  # beta   – dianlight/hassio-addons-beta
    "local_sambanas2",     # local development
]

# WebSocket reconnection settings
WS_RECONNECT_INTERVAL = 5  # seconds
WS_MAX_RECONNECT_ATTEMPTS = 0  # 0 = unlimited

ATTR_FRIENDLY_NAME = "friendly_name"
ATTR_ICON = "icon"
ATTR_DEVICE_CLASS = "device_class"
