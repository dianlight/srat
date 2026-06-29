//go:build darwin

package sysfs

import "errors"

var errSysfsUnsupported = errors.New("sysfs is unavailable on darwin")

type FS struct{}

func NewFS(mountPoint string) (FS, error) {
	return FS{}, nil
}

type NetClassIface struct {
	Name             string
	AddrAssignType   *int64
	AddrLen          *int64
	Address          string
	Broadcast        string
	Carrier          *int64
	CarrierChanges   *int64
	CarrierUpCount   *int64
	CarrierDownCount *int64
	DevID            *int64
	Dormant          *int64
	Duplex           string
	Flags            *int64
	IfAlias          string
	IfIndex          *int64
	IfLink           *int64
	LinkMode         *int64
	MTU              *int64
	NameAssignType   *int64
	NetDevGroup      *int64
	OperState        string
	PhysPortID       string
	PhysPortName     string
	PhysSwitchID     string
	Speed            *int64
	TxQueueLen       *int64
	Type             *int64
}

type NetClass map[string]NetClassIface

func (fs FS) NetClassByIface(devicePath string) (*NetClassIface, error) {
	return nil, errSysfsUnsupported
}

func (fs FS) NetClassDevices() ([]string, error) {
	return []string{}, nil
}

func (fs FS) NetClass() (NetClass, error) {
	return NetClass{}, nil
}
