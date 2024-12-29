#!/bin/bash

losetup -a
find  /workspaces/srat/backend/test/data/ -name "*.dmg" -exec losetup -P -f  '{}' \;
mdev -s
