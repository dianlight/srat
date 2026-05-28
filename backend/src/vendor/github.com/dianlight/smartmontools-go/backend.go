package smartmontools

import smtypes "github.com/dianlight/smartmontools-go/types"

// Backend is the pluggable execution interface for SMART operations.
type Backend = smtypes.Backend

// DiscoveryBackend extends Backend with richer device discovery details.
type DiscoveryBackend = smtypes.DiscoveryBackend
