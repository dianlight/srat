package dto

// FilesystemSupport contains information about filesystem support on the system
type FilesystemSupport struct {
	// CanMount indicates if the filesystem can be mounted
	CanMount bool `json:"canMount"`

	// CanFormat indicates if the filesystem can be formatted (mkfs available)
	CanFormat bool `json:"canFormat"`

	// CanCheck indicates if the filesystem can be checked (fsck available)
	CanCheck bool `json:"canCheck"`

	// CanSetLabel indicates if the filesystem label can be changed
	CanSetLabel bool `json:"canSetLabel"`

	// CanGetState indicates if filesystem state can be retrieved
	CanGetState bool `json:"canGetState"`

	// AlpinePackage is the Alpine Linux package name that provides support
	AlpinePackage string `json:"alpinePackage,omitempty"`

	// MissingTools lists the tools that are not available
	MissingTools []string `json:"missingTools,omitempty"`
}

// FormatOptions contains options for formatting a filesystem
type FormatOptions struct {
	// Label is the filesystem label to set during formatting
	Label string `json:"label,omitempty"`

	// Force forces formatting even if the device appears to be in use
	Force bool `json:"force,omitempty"`

	// AdditionalOptions contains filesystem-specific options
	AdditionalOptions map[string]string `json:"additionalOptions,omitempty"`
}

// CheckOptions contains options for checking a filesystem
type CheckOptions struct {
	// AutoFix automatically fixes errors if possible
	AutoFix bool `json:"autoFix,omitempty"`

	// Force forces check even if filesystem appears clean
	Force bool `json:"force,omitempty"`

	// Verbose enables verbose output
	Verbose bool `json:"verbose,omitempty"`
}

// CheckResult contains the result of a filesystem check operation
type CheckResult struct {
	// Success indicates if the check completed successfully
	Success bool `json:"success"`

	// ErrorsFound indicates if errors were found
	ErrorsFound bool `json:"errorsFound"`

	// ErrorsFixed indicates if errors were fixed (when AutoFix is enabled)
	ErrorsFixed bool `json:"errorsFixed"`

	// Message contains a human-readable message about the check result
	Message string `json:"message,omitempty"`

	// ExitCode is the exit code from the check command
	ExitCode int `json:"exitCode"`
}

// FilesystemState represents the current state of a filesystem
type FilesystemState struct {
	// IsClean indicates if the filesystem is in a clean state
	IsClean bool `json:"isClean"`

	// IsMounted indicates if the filesystem is currently mounted
	IsMounted bool `json:"isMounted"`

	// HasErrors indicates if the filesystem has errors
	HasErrors bool `json:"hasErrors"`

	// StateDescription is a human-readable description of the state
	StateDescription string `json:"stateDescription,omitempty"`

	// AdditionalInfo contains filesystem-specific state information
	AdditionalInfo map[string]interface{} `json:"additionalInfo,omitempty"`
}

// FsMagicSignature defines a structure to hold filesystem signature information
type FsMagicSignature struct {
	// Offset is the byte offset where the magic signature is located
	Offset int64 `json:"offset"`

	// Magic is the byte sequence that identifies the filesystem
	Magic []byte `json:"magic"`
}

// FilesystemTask represents data for filesystem operations (format, check)
type FilesystemTask struct {
	// Device is the device path being operated on
	Device string `json:"device"`

	// Operation is the type of operation ("format" or "check")
	Operation string `json:"operation"`

	// FilesystemType is the filesystem type being used
	FilesystemType string `json:"filesystemType,omitempty"`

	// Status is the current status ("start", "success", "failure")
	Status string `json:"status"`

	// Message provides additional context about the operation
	Message string `json:"message,omitempty"`

	// Error contains error details if status is "failure"
	Error string `json:"error,omitempty"`

	// Result contains operation result details (for success status)
	Result interface{} `json:"result,omitempty"`
}
