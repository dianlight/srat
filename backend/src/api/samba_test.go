// endpoints_test.go
package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/dianlight/srat/config"
	"github.com/gorilla/mux"
)

func TestApplySambaHandler(t *testing.T) {

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequestWithContext(testContext, "POST", "/samba/apply", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/samba/apply", ApplySamba).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNoContent && status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v or %v",
			status, http.StatusNoContent, http.StatusInternalServerError)
	}
}

func checkStringInSMBConfig(testvalue string, expected string, t *testing.T) bool {
	stream, err := createConfigStream(testContext)
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
	stream, err := createConfigStream(testContext)
	if err != nil {
		t.Errorf("Error in createConfigStream %s", err.Error())
	}
	if stream == nil {
		t.Errorf("handler returned stream is nil")
	}

	addon_config := testContext.Value("addon_config").(*config.Config)

	addon_config.Workgroup = "WORKGROUP12"
	if checkStringInSMBConfig(addon_config.Workgroup, "\n\\s*workgroup = *%s\\s*\n", t) == false {
		t.Errorf("Seting workgroup failed.")
	}

	addon_config.Username = "admin"
	if checkStringInSMBConfig(addon_config.Username, "\n\\s*valid users =_ha_mount_user_ %s\\s*\n", t) == false {
		t.Errorf("Seting username failed.")
	}

	addon_config.Moredisks = []string{"ALPHA", "beta"}
	addon_config.Shares["ALPHA"] = config.Share{Path: "/mnt/ALPHA", FS: "ext4"}
	addon_config.Shares["BETA"] = config.Share{Path: "/mnt/BETA", FS: "ext4"}
	if checkStringInSMBConfig(addon_config.Moredisks[0], "\n.*.shares=map[%s:map[fs:ext4 path:/mnt/ALPHA].*\n", t) == false {
		t.Errorf("Setting moredisks and share failed.")
	}

	addon_config.Medialibrary.Enable = true
	addon_config.ACL = []config.OptionsAcl{{Share: "APLHA", Usage: "media", Users: []string{"admin"}, Disabled: false}}

	addon_config.AllowHost = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fe80::/10"}
	addon_config.VetoFiles = []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"}

	addon_config.CompatibilityMode = false
	addon_config.EnableRecycleBin = false

	addon_config.WSDD = false
	addon_config.WSDD2 = false

	//addon_config.OtherUsers = []User{{Username: "test", Password: "test"}, {Username: "test2", Password: "test2"}}
	addon_config.ACL = []config.OptionsAcl{{Share: "config", Disabled: true}}
	addon_config.Interfaces = []string{"eth0"}
	addon_config.BindAllInterfaces = false
	addon_config.LogLevel = "info"
	addon_config.MultiChannel = false

	// Skip because untestable on config file
	//  addon_config.Password = "admin"
	// 	addon_config.Automount = true
	// 	addon_config.AvailableDiskLog = true
	//  addon_config.HDDIdle = 0
	//	addon_config.Smart = false
	//  addon_config.MQTTNextGen = false
	//  addon_config.MQTTEnable = false
	//  addon_config.MQTTHost = ""
	//  addon_config.MQTTUsername = ""
	//  addon_config.MQTTPassword = ""
	//  addon_config.MQTTPort = ""
	//  addon_config.MQTTTopic = ""
	// 	addon_config.Autodiscovery.DisableAutoremove = false
	// 	addon_config.Autodiscovery.DisableDiscovery = false
	// 	addon_config.Autodiscovery.DisablePersistent = false
	//  addon_config.MOF = "42"
	//  addon_config.Mountoptions = []string{"uid=1999, gid=1000, umask=000,iocharset=utf8"}
	//  addon_config.Medialibrary.SSHKEY = "<super secret key>"

}

// check migrate config don't duplicate share

func TestGetSambaProcessStatus(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequestWithContext(testContext, "GET", "/samba/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetSambaProcessStatus)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v or %v",
			status, http.StatusOK, http.StatusNotFound)
	}
}
