// Code generated by github.com/jmattheis/goverter, DO NOT EDIT.
//go:build !goverter

package converter

import (
	dto "github.com/dianlight/srat/dto"
	net "github.com/jaypipes/ghw/pkg/net"
)

type NetToDtoImpl struct{}

func (c *NetToDtoImpl) NetInfoToNetworkInfo(source net.Info, target *dto.NetworkInfo) error {
	if source.NICs != nil {
		target.NICs = make([]dto.NIC, len(source.NICs))
		for i := 0; i < len(source.NICs); i++ {
			target.NICs[i] = c.pNetNICToDtoNIC(source.NICs[i])
		}
	}
	return nil
}
func (c *NetToDtoImpl) pNetNICToDtoNIC(source *net.NIC) dto.NIC {
	var dtoNIC dto.NIC
	if source != nil {
		var dtoNIC2 dto.NIC
		dtoNIC2.Name = (*source).Name
		dtoNIC2.MACAddress = (*source).MACAddress
		dtoNIC2.IsVirtual = (*source).IsVirtual
		dtoNIC2.Speed = (*source).Speed
		dtoNIC2.Duplex = (*source).Duplex
		dtoNIC = dtoNIC2
	}
	return dtoNIC
}
