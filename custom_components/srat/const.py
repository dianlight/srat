"""Constants for the SRAT integration."""

DOMAIN = "srat"

CONF_HOST = "host"
CONF_PORT = "port"
CONF_ADDON_SLUG = "addon_slug"

DEFAULT_PORT = 8099
DEFAULT_HOST = "localhost"

# Addon slugs that can be auto-discovered via the Supervisor API
ADDON_SLUG_WHITELIST = [
    "c751bc52_srat",
    "c751bc52_samba_nas",
    "local_srat",
    "local_samba_nas",
]

# WebSocket reconnection settings
WS_RECONNECT_INTERVAL = 5  # seconds
WS_MAX_RECONNECT_ATTEMPTS = 0  # 0 = unlimited

# Sensor update interval
SENSOR_UPDATE_INTERVAL = 30  # seconds

ATTR_FRIENDLY_NAME = "friendly_name"
ATTR_ICON = "icon"
ATTR_DEVICE_CLASS = "device_class"
