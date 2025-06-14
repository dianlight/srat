package dto

type BinaryAsset struct {
	Name               string `json:"name"`
	Size               int    `json:"size"`
	ID                 int64  `json:"id"`
	BrowserDownloadURL string `json:"browser_download_url,omitempty"`
}

type ReleaseAsset struct {
	LastRelease string      `json:"last_release,omitempty"`
	ArchAsset   BinaryAsset `json:"arch_asset,omitempty"`
}

type UpdateProgress struct {
	ProgressStatus UpdateProcessState `json:"update_process_state,omitempty" enum:"Idle,Checking,NoUpgrde,UpgradeAvailable,Downloading,DownloadComplete,Extracting,ExtractComplete,Installing,NeedRestart,Error"`
	Progress       int                `json:"progress,omitempty"`
	LastRelease    string             `json:"last_release,omitempty"`
	ErrorMessage   string             `json:"error_message,omitempty"`
}
