#!/bin/bash
echo "----------------------------------------------------------------"
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
echo "$SCRIPT_DIR"
losetup -a ||:
find ${SCRIPT_DIR}/../backend/test/data/ -name "*.dmg" -exec losetup -P -f '{}' \; -exec losetup -f \; ||:
mdev -s 2>/dev/null ||:
losetup -a | grep .devcontainer ||:
echo "----------------------------------------------------------------"
for i in "config" "addons" "ssl" "share" "backup" "media" "addon_configs"; do
    mkdir -p /$i ||:
    mount -o bind,ro ${SCRIPT_DIR}/../backend/test/ /$i ||:
done