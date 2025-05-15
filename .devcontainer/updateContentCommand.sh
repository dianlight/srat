#!/bin/bash

apk add --no-cache git make lsblk eudev gcc musl-dev linux-headers samba ethtool e2fsprogs e2fsprogs-extra fuse3 exfatprogs ntfs-3g-progs apfs-fuse openssh-client sshfs pre-commit
apk add --no-cache --update-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community  go "go~=1.24" 
#bun
curl -fsSL https://bun.sh/install | bash -s "bun-v1.2.13" 


make -C .. prepare
