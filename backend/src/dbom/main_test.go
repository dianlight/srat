package dbom

import (
	"log"
	"os"
	"testing"

	"github.com/snapcore/snapd/osutil"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	InitDB("file::memory:?cache=shared&_pragma=foreign_keys(1)")
	//InitDB("test.db?cache=shared&_pragma=foreign_keys=ON")
	data, err := os.ReadFile("../../test/data/mount_info.txt")
	if err != nil {
		log.Fatal(err)
	}
	osutil.MockMountInfo(string(data))
	os.Exit(m.Run())
}
