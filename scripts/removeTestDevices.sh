#!/bin/bash

# This script removes all loop devices.

# Check if root privileges are available.
if [[ $EUID -ne 0 ]]; then
  echo "This script must be run as root."
  exit 1
fi

# Get a list of all loop devices.
loop_devices=$(losetup -a | awk '{print $1}' | cut -d: -f1)

# Remove each loop device.
for device in $loop_devices; do
  losetup -d "$device"
done

echo "All loop devices removed successfully."