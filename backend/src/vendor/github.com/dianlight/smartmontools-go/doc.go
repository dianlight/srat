/*
Package smartmontools provides Go bindings for interfacing with smartmontools
to monitor and manage storage device health using S.M.A.R.T. data.

The library wraps the smartctl command-line utility and provides a clean,
idiomatic Go API for accessing SMART information from storage devices.

# Features

  - Device scanning and discovery
  - SMART health status checking
  - Detailed SMART attribute reading
  - Temperature monitoring
  - Power-on time tracking
  - Self-test execution and progress monitoring
  - Device information retrieval
  - SMART support detection and management
  - Self-test availability checking

# Prerequisites

This library requires smartctl (part of smartmontools) to be installed:

Linux:

	sudo apt-get install smartmontools  # Debian/Ubuntu
	sudo yum install smartmontools       # RHEL/CentOS/Fedora
	sudo pacman -S smartmontools         # Arch Linux

macOS:

	brew install smartmontools

Windows:

	Download from https://www.smartmontools.org/

# Basic Usage

	package main

	import (
	    "fmt"
	    "log"

	    "github.com/dianlight/smartmontools-go"
	)

	func main() {
	    // Create a new client
	    client, err := smartmontools.NewClient()
	    if err != nil {
	        log.Fatal(err)
	    }

	    // Scan for devices
	    devices, err := client.ScanDevices()
	    if err != nil {
	        log.Fatal(err)
	    }

	    // Check health of first device
	    if len(devices) > 0 {
	        healthy, _ := client.CheckHealth(devices[0].Name)
	        if healthy {
	            fmt.Println("Device is healthy")
	        }
	    }
	}

# Permissions

Many operations require elevated privileges (root/administrator) to access
disk devices. The library will return errors if permissions are insufficient.

# Thread Safety

The Client type is safe for concurrent use by multiple goroutines.
*/
package smartmontools
