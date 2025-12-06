package dto

type DataDirtyTracker struct {
	Shares   bool `json:"shares"`
	Users    bool `json:"users"`
	Settings bool `json:"settings"`
}
