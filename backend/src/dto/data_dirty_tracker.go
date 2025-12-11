package dto

type DataDirtyTracker struct {
	Shares   bool `json:"shares"`
	Users    bool `json:"users"`
	Settings bool `json:"settings"`
}

func (d *DataDirtyTracker) IsDirty() bool {
	return d.Shares || d.Users || d.Settings
}

func (d *DataDirtyTracker) MarkAllClean() {
	d.Shares = false
	d.Users = false
	d.Settings = false
}

func (d *DataDirtyTracker) AndMask(tg DataDirtyTracker) bool {
	return (d.Shares && tg.Shares) ||
		(d.Users && tg.Users) ||
		(d.Settings && tg.Settings)
}
