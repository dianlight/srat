#!/usr/bin/env bash
set -x

# Simulated disk device for testing
mkdir -p /dev/disk/by-id
ln -sf ../../vdb1 /dev/disk/by-id/1234-5678

# Attach test image as loop device and mount HA directory structure
losetup -f /workspaces/srat/backend/test/data/image.dmg
for dir in /backup /config /ssl /addon_configs /addons /share /media; do
	if [ ! -d "$dir" ]; then
		mkdir -p "$dir"
		mount /dev/loop0 "$dir"
	fi
done
