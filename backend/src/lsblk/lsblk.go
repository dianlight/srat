package lsblk

// Based on the code found at https://github.com/cedarwu/lsblk/tree/master

import (
	"encoding/json"
	"errors"
	"log"
	"os/exec"
	"strings"
)

type Device struct {
	Name        string   `json:"name,omitempty"`
	Path        string   `json:"path,omitempty"`
	Fsavail     uint64   `json:"fsavail,omitempty" copier:"-"`
	Fssize      uint64   `json:"fssize,omitempty" copier:"-"`
	Fsused      uint64   `json:"fsused,omitempty" copier:"-"`
	Fsusage     uint     `json:"fsusage,omitempty"` // percent that was used
	Fstype      string   `json:"fstype,omitempty"`
	Pttype      string   `json:"pttype,omitempty"`
	Mountpoint  string   `json:"mountpoint,omitempty"`
	Mountpoints []string `json:"mountpoints,omitempty"`
	Label       string   `json:"label,omitempty"`
	UUID        string   `json:"uuid,omitempty"`
	Removable   bool     `json:"rm,omitempty"`
	Hotplug     bool     `json:"hotplug,omitempty"`
	Serial      string   `json:"serial,omitempty"`
	State       string   `json:"state,omitempty"`
	Group       string   `json:"group,omitempty"`
	Type        string   `json:"type,omitempty"`
	//	Alignment  int      `json:"alignment"`
	Wwn        string   `json:"wwn,omitempty"`
	Hctl       string   `json:"hctl,omitempty"`
	Tran       string   `json:"tran,omitempty"`
	Subsystems string   `json:"subsystems,omitempty"`
	Rev        string   `json:"rev,omitempty"`
	Vendor     string   `json:"vendor,omitempty"`
	Model      string   `json:"model,omitempty"`
	Children   []Device `json:"children,omitempty"`
	Partlabel  string   `json:"partlabel,omitempty"`
	Parttype   string   `json:"parttype,omitempty"`
	Partuuid   string   `json:"partuuid,omitempty"`
	Ptuuid     string   `json:"ptuuid,omitempty"`
	ReadOnly   bool     `json:"ro,omitempty"`
}

type _Device struct {
	Name        string      `json:"name"`
	Path        string      `json:"path"`
	Fsavail     interface{} `json:"fsavail"`
	Fssize      interface{} `json:"fssize"`
	Fstype      string      `json:"fstype"`
	Pttype      string      `json:"pttype"`
	Fsused      interface{} `json:"fsused"`
	Fsuse       string      `json:"fsuse%"`
	Mountpoint  string      `json:"mountpoint"`
	Mountpoints []string    `json:"mountpoints"`
	Label       string      `json:"label"`
	UUID        string      `json:"uuid"`
	Removable   bool        `json:"rm"`
	Hotplug     bool        `json:"hotplug"`
	Serial      string      `json:"serial"`
	State       string      `json:"state"`
	Group       string      `json:"group"`
	Type        string      `json:"type"`
	//	Alignment  int         `json:"alignment"`
	Wwn        string    `json:"wwn"`
	Hctl       string    `json:"hctl"`
	Tran       string    `json:"tran"`
	Subsystems string    `json:"subsystems"`
	Rev        string    `json:"rev"`
	Vendor     string    `json:"vendor"`
	Model      string    `json:"model"`
	Children   []_Device `json:"children"`
	Partlabel  string    `json:"partlabel"`
	Parttype   string    `json:"parttype"`
	Partuuid   string    `json:"partuuid"`
	Ptuuid     string    `json:"ptuuid"`
	ReadOnly   bool      `json:"ro"`
}

func runCmd(command string) (output []byte, err error) {
	if len(command) == 0 {
		return nil, errors.New("invalid command")
	}
	commands := strings.Fields(command)
	output, err = exec.Command(commands[0], commands[1:]...).Output()
	return output, err
}

/*

func runBash(command string) (output []byte, err error) {
	if len(command) == 0 {
		return nil, errors.New("invalid command")
	}
	output, err = exec.Command("bash", "-c", command).Output()
	return output, err
}

func PrintDevices(devices map[string]Device) {
	var devList []Device
	for _, dev := range devices {
		devList = append(devList, dev)
	}
	sort.Slice(devList, func(i, j int) bool {
		return devList[i].Name < devList[j].Name
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"name", "hctl", "fstype", "fssize", "fsused", "fsavail", "fsuse%", "type", "mount", "pttype", "vendor", "model"})

	for _, dev := range devList {
		table.Append([]string{dev.Name, dev.Hctl, dev.Fstype, humanize.Bytes(dev.Fssize), humanize.Bytes(dev.Fsused), humanize.Bytes(dev.Fsavail), strconv.FormatUint(uint64(dev.Fsusage), 10) + "%", dev.Type, dev.Mountpoint, dev.Pttype, dev.Vendor, dev.Model})
	}
	table.Render() // Send output
}

func PrintPartitions(devices map[string]Device) {
	partDevMap := make(map[string]string)
	var partList []Device
	for _, dev := range devices {
		for _, child := range dev.Children {
			partDevMap[child.Name] = dev.Name
			child.Vendor = dev.Vendor
			child.Model = dev.Model
			partList = append(partList, child)
		}
	}
	sort.Slice(partList, func(i, j int) bool {
		return partList[i].Name < partList[j].Name
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"disk", "partition", "label", "fstype", "fssize", "fsused", "fsavail", "fsuse%", "type", "mount", "pttype", "vendor", "model"})

	for _, part := range partList {
		table.Append([]string{partDevMap[part.Name], part.Name, part.Label, part.Fstype, humanize.Bytes(part.Fssize), humanize.Bytes(part.Fsused), humanize.Bytes(part.Fsavail), strconv.FormatUint(uint64(part.Fsusage), 10) + "%", part.Type, part.Mountpoint, part.Pttype, part.Vendor, part.Model})
	}
	table.Render() // Send output
}

func create(x uint64) *uint64 {
	return &x
}

func allToUint64WithDefault(in interface{}, def uint64) *uint64 {
	switch v := in.(type) {
	case nil:
		return &def
	case float64:
		return create(uint64(v))
	case int, int64, uint64:
		return create(uint64(v.(uint64)))
	case string:
		val, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			return &val
		}
		return &def
	default:
		return &def
	}
}

func ListDevices() (devices map[string]Device, err error) {
	//	output, err := runCmd("lsblk -e7 -b -J -o name,path,fsavail,fssize,fstype,pttype,fsused,fsuse%,mountpoint,label,uuid,rm,hotplug,serial,state,group,type,alignment,wwn,hctl,tran,subsystems,rev,vendor,model")
	output, err := runCmd("lsblk -e7 -b -J -o name,path,fsavail,fssize,fstype,pttype,fsused,fsuse%,mountpoint,mountpoints,label,uuid,rm,hotplug,serial,state,group,type,wwn,hctl,tran,subsystems,rev,vendor,model,partlabel,parttype,partuuid,ptuuid,ro")
	if err != nil {
		return nil, err
	}

	lsblkRsp := make(map[string][]_Device)
	err = json.Unmarshal(output, &lsblkRsp)
	if err != nil {
		log.Println(output)
		return nil, err
	}

	devices = make(map[string]Device)
	for _, _device := range lsblkRsp["blockdevices"] {
		var device Device
		var err = copier.Copy(&device, &_device)
		if err != nil {
			log.Println(err)
		}
		//log.Println(len(device.Children), len(_device.Children))

		device.Fsavail = *allToUint64WithDefault(_device.Fsavail, 0)
		device.Fssize = *allToUint64WithDefault(_device.Fssize, 0)
		device.Fsused = *allToUint64WithDefault(_device.Fsused, 0)

		if device.Fssize > 0 {
			device.Fsusage = uint(math.Round(float64(device.Fsused*100) / float64(device.Fssize)))
		}

		for i, child := range _device.Children {
			//log.Println(i, len(device.Children), child, *allToUint64WithDefault(child.Fsavail, 0))

			device.Children[i].Fsavail = *allToUint64WithDefault(child.Fsavail, 0)
			device.Children[i].Fsused = *allToUint64WithDefault(child.Fsused, 0)
			device.Children[i].Fssize = *allToUint64WithDefault(child.Fssize, 0)
			if device.Children[i].Fssize > 0 {
				device.Children[i].Fsusage = uint(math.Round(float64(device.Children[i].Fsused*100) / float64(device.Children[i].Fssize)))
			}
		}

		serial, err := getSerial(device.Name)
		if err == nil {
			device.Serial = serial
		}
		devices[device.Name] = device
	}

	return devices, nil
}

func getSerial(devName string) (serial string, err error) {
	output, err := runBash("udevadm info --query=property --name=/dev/" + devName + " | grep SCSI_IDENT_SERIAL | awk -F'=' '{print $2}'")
	return strings.TrimSpace(string(output)), err
}
*/

func GetLabelsFromDevice(devName string) (labels *string, partlabel *string, mountpoint *string, err error) {
	output, err := runCmd("lsblk -e7 -b -J -o name,label,partlabel,mountpoint /dev/" + devName)
	if err != nil {
		return nil, nil, nil, err
	}

	lsblkRsp := make(map[string][]_Device)
	err = json.Unmarshal(output, &lsblkRsp)
	if err != nil {
		log.Println(output)
		return nil, nil, nil, err
	}
	return &lsblkRsp["blockdevices"][0].Label, &lsblkRsp["blockdevices"][0].Partlabel, &lsblkRsp["blockdevices"][0].Mountpoint, nil
}
