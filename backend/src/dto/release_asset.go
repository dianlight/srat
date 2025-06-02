package dto

type BinaryAsset struct {
	Size               int    `json:"size"`
	ID                 int64  `json:"id"`
	BrowserDownloadURL string `json:"browser_download_url,omitempty"`
}

type ReleaseAsset struct {
	LastRelease string      `json:"last_release,omitempty"`
	ArchAsset   BinaryAsset `json:"arch_asset,omitempty"`
}

type UpdateProgress struct {
	ProgressStatus int8   `json:"update_status"`
	LastRelease    string `json:"last_release,omitempty"`
	UpdateError    string `json:"update_error,omitempty"`
}
