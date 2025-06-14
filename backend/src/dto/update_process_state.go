package dto

//go:generate go tool goenums update_process_state.go
type updateProcessState int8 // Name[string]

const (
	UpdateStatusIdle             updateProcessState = iota // "Idle"
	UpdateStatusChecking                                   // "Checking"
	UpdateStatusNoUpgrde                                   // "NoUpgrade"
	UpdateStatusUpgradeAvailable                           // "Available"
	UpdateStatusDownloading                                // "Downloading"
	UpdateStatusDownloadComplete                           // "Downloaded"
	UpdateStatusExtracting                                 // "Extractiong"
	UpdateStatusExtractComplete                            // "Extracted"
	UpdateStatusInstalling                                 // "Installing"
	UpdateStatusInstallComplete                            // "NeedRestart" (Ready for restart)
	UpdateStatusError                                      // "Error"
)
