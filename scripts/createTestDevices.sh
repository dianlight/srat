#!/bin/bash
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
losetup -a
find  $(SCRIPT_DIR)/../backend/test/data/ -name "*.dmg" -exec losetup -P -f  '{}' \;
mdev -s
