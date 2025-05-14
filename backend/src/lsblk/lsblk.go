package lsblk

// Based on the code found at https://github.com/cedarwu/lsblk/tree/master

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/itchyny/gojq"
	"gitlab.com/tozd/go/errors"
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
	Wwn         string   `json:"wwn,omitempty"`
	Hctl        string   `json:"hctl,omitempty"`
	Tran        string   `json:"tran,omitempty"`
	Subsystems  string   `json:"subsystems,omitempty"`
	Rev         string   `json:"rev,omitempty"`
	Vendor      string   `json:"vendor,omitempty"`
	Model       string   `json:"model,omitempty"`
	Children    []Device `json:"children,omitempty"`
	Partlabel   string   `json:"partlabel,omitempty"`
	Parttype    string   `json:"parttype,omitempty"`
	Partuuid    string   `json:"partuuid,omitempty"`
	Ptuuid      string   `json:"ptuuid,omitempty"`
	ReadOnly    bool     `json:"ro,omitempty"`
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
	Wwn         string      `json:"wwn"`
	Hctl        string      `json:"hctl"`
	Tran        string      `json:"tran"`
	Subsystems  string      `json:"subsystems"`
	Rev         string      `json:"rev"`
	Vendor      string      `json:"vendor"`
	Model       string      `json:"model"`
	Children    []_Device   `json:"children"`
	Partlabel   string      `json:"partlabel"`
	Parttype    string      `json:"parttype"`
	Partuuid    string      `json:"partuuid"`
	Ptuuid      string      `json:"ptuuid"`
	ReadOnly    bool        `json:"ro"`
}

type LSBKInfo struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Partlabel  string `json:"partlabel"`
	Mountpoint string `json:"mountpoint"`
	Fstype     string `json:"fstype"`
}

var _lsbkInterpreterInstance LSBLKInterpreterInterface
var _lsbkInterpreterInstanceMutex sync.Mutex

type LSBLKInterpreterInterface interface {
	GetInfoFromDevice(devName string) (info *LSBKInfo, err error)
}

type LSBKInterpreter struct {
	jqpartition *gojq.Code
	jqdevice    *gojq.Code
}

var DeviceNotFound = errors.Base("device not found")

func NewLSBKInterpreter() LSBLKInterpreterInterface {
	_lsbkInterpreterInstanceMutex.Lock()
	defer _lsbkInterpreterInstanceMutex.Unlock()
	if _lsbkInterpreterInstance != nil {
		return _lsbkInterpreterInstance
	}
	self := &LSBKInterpreter{}
	_lsbkInterpreterInstance = self

	query, err := gojq.Parse(".blockdevices[]| select(.children) | .children[] | select(.name == $p )")
	if err != nil {
		log.Fatalln(err)
	}
	jq, err := gojq.Compile(query,
		gojq.WithVariables([]string{
			"$p",
		}))
	if err != nil {
		log.Fatalln(err)
	}
	self.jqpartition = jq
	query2, err := gojq.Parse(".blockdevices[]| select(.name == $p )")
	if err != nil {
		log.Fatalln(err)
	}
	jq2, err := gojq.Compile(query2,
		gojq.WithVariables([]string{
			"$p",
		}))
	if err != nil {
		log.Fatalln(err)
	}
	self.jqdevice = jq2

	return _lsbkInterpreterInstance
}

func (*LSBKInterpreter) runCmd(command string) (output []byte, err error) {
	if len(command) == 0 {
		return nil, errors.New("invalid command")
	}
	commands := strings.Fields(command)
	output, err = exec.Command(commands[0], commands[1:]...).Output()
	return output, err
}

func (self *LSBKInterpreter) GetInfoFromDevice(devName string) (info *LSBKInfo, err error) {

	devName, _ = strings.CutPrefix(devName, "/dev/")

	// check if file devName exists using os.OpenFile
	if _, err := os.OpenFile(fmt.Sprintf("/dev/%s", devName), os.O_RDONLY, 0); err != nil {
		return nil, errors.Wrap(DeviceNotFound, fmt.Sprintf("device %s not found", devName))
	}

	result := &LSBKInfo{}

	output, err := self.runCmd("lsblk -b -J -o name,label,partlabel,mountpoint,fstype")
	if err != nil {
		return nil, err
	}

	lsblkRsp := make(map[string]any)
	err = json.Unmarshal(output, &lsblkRsp)
	if err != nil {
		log.Println(output)
		return nil, err
	}

	iter := self.jqpartition.Run(lsblkRsp, devName)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Fatalln(err)
			return nil, err
		}
		//fmt.Printf("%#v\n", v)
		if m := v.(map[string]interface{}); m["name"] == devName {
			result.Name = devName
			var str = fmt.Sprintf("%v", m["label"])
			result.Label = strings.Replace(str, "<nil>", "unknown", 1)
			var str1 = fmt.Sprintf("%v", m["partlabel"])
			result.Partlabel = strings.Replace(str1, "<nil>", "unknown", 1)
			var str2 = fmt.Sprintf("%v", m["mountpoint"])
			result.Mountpoint = strings.Replace(str2, "<nil>", "", 1)
			var str3 = fmt.Sprintf("%v", m["fstype"])
			result.Fstype = strings.Replace(str3, "<nil>", "unknown", 1)
			break
		}
	}

	if result.Name == "" {
		iter := self.jqdevice.Run(lsblkRsp, devName)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				log.Fatalln(err)
				return nil, err
			}
			//fmt.Printf("%#v\n", v)
			if m := v.(map[string]interface{}); m["name"] == devName {
				result.Name = devName
				var str = fmt.Sprintf("%v", m["label"])
				result.Label = strings.Replace(str, "<nil>", "unknown", 1)
				var str1 = fmt.Sprintf("%v", m["partlabel"])
				result.Partlabel = strings.Replace(str1, "<nil>", "unknown", 1)
				var str2 = fmt.Sprintf("%v", m["mountpoint"])
				result.Mountpoint = strings.Replace(str2, "<nil>", "", 1)
				var str3 = fmt.Sprintf("%v", m["fstype"])
				result.Fstype = strings.Replace(str3, "<nil>", "unknown", 1)
				break
			}
		}
	}

	//fmt.Printf("--->\n%v\n", *result)
	if result.Name == "" {
		return nil, fmt.Errorf("device %s not found", devName)
	}
	return result, nil
}
