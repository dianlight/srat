#!/bin/bash
echo "----------------------------------------------------------------"
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
echo "$SCRIPT_DIR"
losetup -a
find ${SCRIPT_DIR}/../backend/test/data/ -name "*.dmg" -exec losetup -P -f '{}' \; -exec sleep 1 \; 
mdev -s |:
losetup -a | grep .devcontainer