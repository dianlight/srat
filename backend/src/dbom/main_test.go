package dbom

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	InitDB("file::memory:?cache=shared&_pragma=foreign_keys(1)")
	//InitDB("/tmp/test.db")
	os.Exit(m.Run())
}
