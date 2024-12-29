#!/bin/bash

losetup -a
find  /workspaces/srat/backend/test/data/ -name "*.img" -exec losetup -P -f  '{}' \;
mdev -s
