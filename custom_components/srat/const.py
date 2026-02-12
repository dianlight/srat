"""Constants for the SRAT integration."""

DOMAIN = "srat"

CONF_HOST = "host"
CONF_PORT = "port"
CONF_ADDON_SLUG = "addon_slug"

DEFAULT_PORT = 8099
DEFAULT_HOST = "localhost"

# Addon slugs that can be auto-discovered via the Supervisor API
# From https://github.com/dianlight/hassio-addons and
# https://github.com/dianlight/hassio-addons-beta repositories
ADDON_SLUG_WHITELIST = [
    "local_sambanas2",
    "sambanas2",
]

# WebSocket reconnection settings
WS_RECONNECT_INTERVAL = 5  # seconds
WS_MAX_RECONNECT_ATTEMPTS = 0  # 0 = unlimited

# Sensor update interval
SENSOR_UPDATE_INTERVAL = 30  # seconds

ATTR_FRIENDLY_NAME = "friendly_name"
ATTR_ICON = "icon"
ATTR_DEVICE_CLASS = "device_class"
