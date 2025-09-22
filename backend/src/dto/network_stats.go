package dto

// NetworkStats represents the network health of the system, including global and per-interface statistics.
type NetworkStats struct {
	PerNicIO []NicIOStats   `json:"perNicIO"`
	Global   GlobalNicStats `json:"global"`
}

// NicIOStats represents the I/O statistics for a single network interface.
type NicIOStats struct {
	DeviceName      string  `json:"deviceName"`
	DeviceMaxSpeed  int64   `json:"deviceMaxSpeed"`
	InboundTraffic  float64 `json:"inboundTraffic"`
	OutboundTraffic float64 `json:"outboundTraffic"`
	IP              string  `json:"ip,omitempty"`
	Netmask         string  `json:"netmask,omitempty"`
}

// GlobalNicStats represents the global network statistics for the system.
type GlobalNicStats struct {
	TotalInboundTraffic  float64 `json:"totalInboundTraffic"`
	TotalOutboundTraffic float64 `json:"totalOutboundTraffic"`
}
