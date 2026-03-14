package dto

type DataDirtyTracker struct {
	Shares   bool `json:"shares"`
	Users    bool `json:"users"`
	Settings bool `json:"settings"`
	AppConfig bool `json:"app_config"`
}

func (d *DataDirtyTracker) IsDirty() bool {
	return d.Shares || d.Users || d.Settings || d.AppConfig
}

func (d *DataDirtyTracker) MarkAllClean() {
	d.Shares = false
	d.Users = false
	d.Settings = false
	d.AppConfig = false
}

func (d *DataDirtyTracker) AndMask(tg DataDirtyTracker) bool {
	return (d.Shares && tg.Shares) ||
		(d.Users && tg.Users) ||
		(d.Settings && tg.Settings) ||
		(d.AppConfig && tg.AppConfig)
}
