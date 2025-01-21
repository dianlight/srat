package dto

type DataDirtyTracker struct {
	Shares   bool `json:"shares"`
	Users    bool `json:"users"`
	Volumes  bool `json:"volumes"`
	Settings bool `json:"settings"`
}
