package config

import (
	"testing"
)

func TestMain(m *testing.M) {
	InitDB(":memory:")
	m.Run()
}
