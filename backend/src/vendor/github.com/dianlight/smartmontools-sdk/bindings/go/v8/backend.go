// SPDX-License-Identifier: GPL-2.0-or-later

package smartmontools

import smtypes "github.com/dianlight/smartmontools-sdk/bindings/go/v8/types"

// Backend is the pluggable execution interface for SMART operations.
type Backend = smtypes.Backend

// DiscoveryBackend extends Backend with richer device discovery details.
type DiscoveryBackend = smtypes.DiscoveryBackend
