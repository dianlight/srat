package dbom

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	InitDB("file::memory:?cache=shared&_pragma=foreign_keys(1)")
	//InitDB("test.db?cache=shared&_pragma=foreign_keys=ON")
	os.Exit(m.Run())
}
