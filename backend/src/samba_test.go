// endpoints_test.go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gorilla/mux"
)

func TestApplySambaHandler(t *testing.T) {

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/samba/apply", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/samba/apply", applySamba).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func checkStringInSMBConfig(testvalue string, expected string, t *testing.T) bool {
	stream := createConfigStream()
	rexpt := fmt.Sprintf(expected, testvalue)

	if stream == nil {
		t.Errorf("handler returned stream is nil")
		return false
	} else {
		m, err := regexp.MatchString(rexpt, string(*stream))
		if err != nil {
			t.Errorf("Error in regexp %s", err.Error())
			return false
		}
		if m == false {
			t.Errorf("Wrong Match `%s` not found in stream \n%s", rexpt, string(*stream))
			return false
		}
	}
	//	} else if strings.Contains(string(*stream), fmt.Sprintf(expected, testvalue)) == false {
	//		t.Errorf("Wrong Match `%s` not found in stream \n%s", fmt.Sprintf(expected, testvalue), string(*stream))
	//		return false
	//	}
	return true
}

func TestCreateConfigStream(t *testing.T) {
	stream := createConfigStream()
	if stream == nil {
		t.Errorf("handler returned stream is nil")
	}

	config.Workgroup = "WORKGROUP12"
	if checkStringInSMBConfig(config.Workgroup, "\n\\s*workgroup = *%s\\s*\n", t) == false {
		t.Errorf("Seting workgroup failed.")
	}

	config.Username = "admin"
	if checkStringInSMBConfig(config.Username, "\n\\s*valid users =_ha_mount_user_ %s\\s*\n", t) == false {
		t.Errorf("Seting username failed.")
	}

	config.Moredisks = []string{"ALPHA", "beta"}
	config.Shares["ALPHA"] = Share{Path: "/mnt/ALPHA", FS: "ext4"}
	config.Shares["BETA"] = Share{Path: "/mnt/BETA", FS: "ext4"}
	if checkStringInSMBConfig(config.Moredisks[0], "\n.*.shares=map[%s:map[fs:ext4 path:/mnt/ALPHA].*\n", t) == false {
		t.Errorf("Setting moredisks and share failed.")
	}

	config.Medialibrary.Enable = true
	config.ACL = []OptionsAcl{{Share: "APLHA", Usage: "media", Users: []string{"admin"}, Disabled: false}}

	config.AllowHost = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fe80::/10"}
	config.VetoFiles = []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"}

	config.CompatibilityMode = false
	config.EnableRecycleBin = false

	config.WSDD = false
	config.WSDD2 = false

	//config.OtherUsers = []User{{Username: "test", Password: "test"}, {Username: "test2", Password: "test2"}}
	config.ACL = []OptionsAcl{{Share: "config", Disabled: true}}
	config.Interfaces = []string{"eth0"}
	config.BindAllInterfaces = false
	config.LogLevel = "info"
	config.MultiChannel = false

	// Skip because untestable on config file
	//  config.Password = "admin"
	// 	config.Automount = true
	// 	config.AvailableDiskLog = true
	//  config.HDDIdle = 0
	//	config.Smart = false
	//  config.MQTTNextGen = false
	//  config.MQTTEnable = false
	//  config.MQTTHost = ""
	//  config.MQTTUsername = ""
	//  config.MQTTPassword = ""
	//  config.MQTTPort = ""
	//  config.MQTTTopic = ""
	// 	config.Autodiscovery.DisableAutoremove = false
	// 	config.Autodiscovery.DisableDiscovery = false
	// 	config.Autodiscovery.DisablePersistent = false
	//  config.MOF = "42"
	//  config.Mountoptions = []string{"uid=1999, gid=1000, umask=000,iocharset=utf8"}
	//  config.Medialibrary.SSHKEY = "<super secret key>"

}
