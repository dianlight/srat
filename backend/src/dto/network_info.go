package dto

type NetworkInfo struct {
	NICs []*NIC `json:"nics"`
}

type NIC struct {
	// Name is the string identifier the system gave this NIC.
	Name string `json:"name"`
	// MACAddress is the Media Access Control (MAC) address of this NIC.
	MACAddress string `json:"mac_address"`
	// IsVirtual is true if the NIC is entirely virtual/emulated, false
	// otherwise.
	IsVirtual bool `json:"is_virtual"`
	// Speed is a string describing the link speed of this NIC, e.g. "1000Mb/s"
	Speed string `json:"speed"`
	// Duplex is a string indicating the current duplex setting of this NIC,
	// e.g. "Full"
	Duplex string `json:"duplex"`
}

/*

func (self *NetworkInfo) From(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self *NetworkInfo) FromIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(self, value, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self NetworkInfo) To(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: false, DeepCopy: true})
}
func (self NetworkInfo) ToIgnoreEmpty(value interface{}) error {
	return copier.CopyWithOption(value, &self, copier.Option{IgnoreEmpty: true, DeepCopy: true})
}
func (self NetworkInfo) ToResponse(code int, w http.ResponseWriter) error {
	return doResponse(code, w, self)
}
func (self NetworkInfo) ToResponseError(code int, w http.ResponseWriter, message string, body any) error {
	return doResponseError(code, w, message, body)
}
func (self *NetworkInfo) FromJSONBody(w http.ResponseWriter, r *http.Request) error {
	return fromJSONBody(w, r, self)
}
*/
