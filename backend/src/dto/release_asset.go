package dto

type BinaryAsset struct {
	Size int   `json:"size"`
	ID   int64 `json:"id"`
	// Arch string `json:"arch"`
}

type ReleaseAsset struct {
	//ProgressStatus int8        `json:"update_status"`
	LastRelease string      `json:"last_release,omitempty"`
	ArchAsset   BinaryAsset `json:"arch_asset,omitempty"`
	//LastRelease  *github.RepositoryRelease `json:"last_release,omitempty"`
	//ArchAsset    *github.ReleaseAsset      `json:"arch,omitempty"`
}

type UpdateProgress struct {
	ProgressStatus int8   `json:"update_status"`
	LastRelease    string `json:"last_release,omitempty"`
	UpdateError    string `json:"update_error,omitempty"`
}
