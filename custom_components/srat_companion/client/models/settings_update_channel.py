from enum import Enum


class SettingsUpdateChannel(str, Enum):
    DEVELOP = "Develop"
    NONE = "None"
    PRERELEASE = "Prerelease"
    RELEASE = "Release"

    def __str__(self) -> str:
        return str(self.value)
