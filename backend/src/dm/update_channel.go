package dm

type UpdateChannel string

const (
	Stable     UpdateChannel = "stable"
	Prerelease UpdateChannel = "prerelease"
	None       UpdateChannel = "none"
)