package dto

//From: https://developers.home-assistant.io/docs/api/supervisor/models#mount

type HAMountUsage string // https://developers.home-assistant.io/docs/api/supervisor/models#mount

const (
	UsageAsNone     HAMountUsage = "none"
	UsageAsBackup   HAMountUsage = "backup"
	UsageAsMedia    HAMountUsage = "media"
	UsageAsShare    HAMountUsage = "share"
	UsageAsInternal HAMountUsage = "internal"
)
