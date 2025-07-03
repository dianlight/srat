from enum import Enum


class UpdateProgressUpdateProcessState(str, Enum):
    CHECKING = "Checking"
    DOWNLOADCOMPLETE = "DownloadComplete"
    DOWNLOADING = "Downloading"
    ERROR = "Error"
    EXTRACTCOMPLETE = "ExtractComplete"
    EXTRACTING = "Extracting"
    IDLE = "Idle"
    INSTALLING = "Installing"
    NEEDRESTART = "NeedRestart"
    NOUPGRDE = "NoUpgrde"
    UPGRADEAVAILABLE = "UpgradeAvailable"

    def __str__(self) -> str:
        return str(self.value)
