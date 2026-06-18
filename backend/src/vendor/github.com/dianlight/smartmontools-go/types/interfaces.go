package types

import "context"

// LogAdapter captures the logging methods used by this package.
// It is satisfied by both *slog.Logger and *tlog.Logger.
type LogAdapter interface {
	Debug(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// Backend is the pluggable execution interface for SMART operations.
type Backend interface {
	Name() string
	ScanDevices(ctx context.Context) ([]Device, error)
	GetSMARTInfo(ctx context.Context, devicePath string) (*SMARTInfo, error)
	CheckHealth(ctx context.Context, devicePath string) (bool, error)
	GetDeviceInfo(ctx context.Context, devicePath string) (map[string]any, error)
	RunSelfTest(ctx context.Context, devicePath string, testType string) error
	GetAvailableSelfTests(ctx context.Context, devicePath string) (*SelfTestInfo, error)
	EnableSMART(ctx context.Context, devicePath string) error
	DisableSMART(ctx context.Context, devicePath string) error
	AbortSelfTest(ctx context.Context, devicePath string) error
	Close() error
}

// DiscoveryBackend is an optional extension of Backend that provides richer
// device discovery with per-device protocol-fallback details.
type DiscoveryBackend interface {
	Backend
	DiscoverDevices(ctx context.Context) ([]DiscoveryResult, error)
}

// Commander is the interface for executing OS commands.
type Commander interface {
	Command(ctx context.Context, logger LogAdapter, name string, arg ...string) Cmd
}

// Cmd is the interface for a running command.
type Cmd interface {
	Output() ([]byte, error)
	Run() error
	CombinedOutput() ([]byte, error)
}
