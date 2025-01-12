package dbom

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	InitDB(":memory:?cache=shared&_pragma=foreign_keys(1)")
	os.Exit(m.Run())
}
