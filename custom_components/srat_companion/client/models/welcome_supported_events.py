from enum import Enum


class WelcomeSupportedEvents(str, Enum):
    DIRTY = "dirty"
    HEARTBEAT = "heartbeat"
    HELLO = "hello"
    SHARE = "share"
    UPDATING = "updating"
    VOLUMES = "volumes"

    def __str__(self) -> str:
        return str(self.value)
