from enum import Enum


class PatchShareByShareNameJsonBodyUsage(str, Enum):
    BACKUP = "backup"
    INTERNAL = "internal"
    MEDIA = "media"
    NONE = "none"
    SHARE = "share"

    def __str__(self) -> str:
        return str(self.value)
