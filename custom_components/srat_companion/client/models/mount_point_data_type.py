from enum import Enum


class MountPointDataType(str, Enum):
    ADDON = "ADDON"
    HOST = "HOST"

    def __str__(self) -> str:
        return str(self.value)
