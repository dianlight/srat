package smartmontools

import (
	"log/slog"

	smcompare "github.com/dianlight/smartmontools-go/backends/compare"
	"github.com/dianlight/tlog"
)

// CompareBackend is a virtual Backend that runs multiple backends in parallel,
// returns the first (master) backend's results, and logs discrepancies.
type CompareBackend = smcompare.CompareBackend

// CompareBackendOption configures a CompareBackend.
type CompareBackendOption = smcompare.Option

// NewCompareBackend creates a CompareBackend with at least two backends.
// The first backend is the master; its results are always returned to the caller.
// Secondary backends run in parallel; result mismatches are logged as warnings
// and secondary errors are logged as errors.
func NewCompareBackend(backends []Backend, opts ...CompareBackendOption) (*CompareBackend, error) {
	return smcompare.NewCompareBackend(backends, opts...)
}

// WithCompareLogHandler sets a custom LogAdapter for a CompareBackend.
func WithCompareLogHandler(logger LogAdapter) CompareBackendOption {
	return smcompare.WithLogHandler(logger)
}

// WithCompareSlogHandler sets a custom slog.Logger for a CompareBackend.
func WithCompareSlogHandler(logger *slog.Logger) CompareBackendOption {
	return smcompare.WithSlogHandler(logger)
}

// WithCompareTLogHandler sets a custom tlog.Logger for a CompareBackend.
func WithCompareTLogHandler(logger *tlog.Logger) CompareBackendOption {
	return smcompare.WithTLogHandler(logger)
}
