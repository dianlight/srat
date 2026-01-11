package dto

type BinaryAsset struct {
	Name               string `json:"name"`
	Size               int    `json:"size"`
	ID                 int64  `json:"id"`
	BrowserDownloadURL string `json:"browser_download_url,omitempty"`
	Digest             string `json:"digest,omitempty"`
}

type ReleaseAsset struct {
	LastRelease string      `json:"last_release,omitempty"`
	ArchAsset   BinaryAsset `json:"arch_asset,omitempty"`
}

type UpdateProgress struct {
	ProgressStatus UpdateProcessState `json:"update_process_state,omitempty" enum:"Idle,Checking,NoUpgrde,UpgradeAvailable,Downloading,DownloadComplete,Extracting,ExtractComplete,Installing,NeedRestart,Error"`
	Progress       float64            `json:"progress,omitempty"`
	ReleaseAsset   *ReleaseAsset      `json:"release_asset,omitempty"`
	ErrorMessage   string             `json:"error_message,omitempty"`
}
