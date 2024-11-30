package main

import "testing"

func TestMain(m *testing.M) {
	// Get config
	config = readConfig("../test/data/config.json")
	// Get options
	options = readOptionsFile("../test/data/options.json")
	m.Run()
}
