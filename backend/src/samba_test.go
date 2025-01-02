// endpoints_test.go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/dianlight/srat/data"
	"github.com/dianlight/srat/dm"
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
	if status := rr.Code; status != http.StatusNoContent && status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v or %v",
			status, http.StatusNoContent, http.StatusInternalServerError)
	}
}

func checkStringInSMBConfig(testvalue string, expected string, t *testing.T) bool {
	stream, err := createConfigStream()
	if err != nil {
		t.Errorf("Error in createConfigStream %s", err.Error())
		return false
	}
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
	stream, err := createConfigStream()
	if err != nil {
		t.Errorf("Error in createConfigStream %s", err.Error())
	}
	if stream == nil {
		t.Errorf("handler returned stream is nil")
	}

	data.Config.Workgroup = "WORKGROUP12"
	if checkStringInSMBConfig(data.Config.Workgroup, "\n\\s*workgroup = *%s\\s*\n", t) == false {
		t.Errorf("Seting workgroup failed.")
	}

	data.Config.Username = "admin"
	if checkStringInSMBConfig(data.Config.Username, "\n\\s*valid users =_ha_mount_user_ %s\\s*\n", t) == false {
		t.Errorf("Seting username failed.")
	}

	data.Config.Moredisks = []string{"ALPHA", "beta"}
	data.Config.Shares["ALPHA"] = config.Share{Path: "/mnt/ALPHA", FS: "ext4"}
	data.Config.Shares["BETA"] = config.Share{Path: "/mnt/BETA", FS: "ext4"}
	if checkStringInSMBConfig(data.Config.Moredisks[0], "\n.*.shares=map[%s:map[fs:ext4 path:/mnt/ALPHA].*\n", t) == false {
		t.Errorf("Setting moredisks and share failed.")
	}

	data.Config.Medialibrary.Enable = true
	data.Config.ACL = []config.OptionsAcl{{Share: "APLHA", Usage: "media", Users: []string{"admin"}, Disabled: false}}

	data.Config.AllowHost = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fe80::/10"}
	data.Config.VetoFiles = []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"}

	data.Config.CompatibilityMode = false
	data.Config.EnableRecycleBin = false

	data.Config.WSDD = false
	data.Config.WSDD2 = false

	//data.Config.OtherUsers = []User{{Username: "test", Password: "test"}, {Username: "test2", Password: "test2"}}
	data.Config.ACL = []config.OptionsAcl{{Share: "config", Disabled: true}}
	data.Config.Interfaces = []string{"eth0"}
	data.Config.BindAllInterfaces = false
	data.Config.LogLevel = "info"
	data.Config.MultiChannel = false

	// Skip because untestable on config file
	//  data.Config.Password = "admin"
	// 	data.Config.Automount = true
	// 	data.Config.AvailableDiskLog = true
	//  data.Config.HDDIdle = 0
	//	data.Config.Smart = false
	//  data.Config.MQTTNextGen = false
	//  data.Config.MQTTEnable = false
	//  data.Config.MQTTHost = ""
	//  data.Config.MQTTUsername = ""
	//  data.Config.MQTTPassword = ""
	//  data.Config.MQTTPort = ""
	//  data.Config.MQTTTopic = ""
	// 	data.Config.Autodiscovery.DisableAutoremove = false
	// 	data.Config.Autodiscovery.DisableDiscovery = false
	// 	data.Config.Autodiscovery.DisablePersistent = false
	//  data.Config.MOF = "42"
	//  data.Config.Mountoptions = []string{"uid=1999, gid=1000, umask=000,iocharset=utf8"}
	//  data.Config.Medialibrary.SSHKEY = "<super secret key>"

}

// check migrate config don't duplicate share

func TestGetSambaProcessStatus(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/samba/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getSambaProcessStatus)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v or %v",
			status, http.StatusOK, http.StatusNotFound)
	}
}

func TestPersistConfig(t *testing.T) {
	// Setup
	//data.Config = &config.Config{}
	data.DirtySectionState = dm.DataDirtyTracker{
		Settings: true,
		Users:    true,
		Shares:   true,
		Volumes:  true,
	}

	// Create a request to pass to our handler
	req, err := http.NewRequest("PUT", "/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(persistConfig)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check if DirtySectionState flags are set to false
	if data.DirtySectionState.Settings {
		t.Errorf("DirtySectionState.Settings not set to false")
	}
	if data.DirtySectionState.Users {
		t.Errorf("DirtySectionState.Users not set to false")
	}
	if data.DirtySectionState.Shares {
		t.Errorf("DirtySectionState.Shares not set to false")
	}
	if data.DirtySectionState.Volumes {
		t.Errorf("DirtySectionState.Volumes not set to false")
	}
}
