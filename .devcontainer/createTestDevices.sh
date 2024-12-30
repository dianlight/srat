#!/bin/bash
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
echo "$SCRIPT_DIR"
losetup -a
find $(SCRIPT_DIR)/../backend/test/data/ -name "*.dmg" -exec losetup -P -f '{}' \;
mdev -s
