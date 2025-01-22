#!/bin/bash
echo "----------------------------------------------------------------"
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
echo "$SCRIPT_DIR"
losetup -a ||:
find ${SCRIPT_DIR}/../backend/test/data/ -name "*.dmg" -exec losetup -P -f '{}' \; -exec sleep 1 \;  ||:
mdev -s 2>/dev/null ||:
losetup -a | grep .devcontainer ||:
echo "----------------------------------------------------------------"
mkdir -p /addons ||:
mount -o bind,ro ${SCRIPT_DIR}/../backend/test/ /addons ||:
mkdir -p /media ||:
mount -o bind,ro ${SCRIPT_DIR}/../backend/test/ /media ||: 