package config

import (
	"context"
	"log"
	"os"
	"testing"
)

var testContext = context.Background()

func TestMain(m *testing.M) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	os.Exit(m.Run())
}
