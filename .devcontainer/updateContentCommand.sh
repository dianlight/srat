#!/bin/bash

apk add --no-cache go git make lsblk eudev gcc musl-dev linux-headers samba ethtool e2fsprogs e2fsprogs-extra fuse3 exfatprogs ntfs-3g-progs apfs-fuse

#bun
curl -fsSL https://bun.sh/install | bash

GOBIN=/usr/local/bin/ go install github.com/rogpeppe/gohack@latest
GOBIN=/usr/local/bin/ go install github.com/rakyll/gotest@latest
GOBIN=/usr/local/bin/ go install github.com/Antonboom/testifylint@latest
GOBIN=/usr/local/bin/ go install github.com/ramya-rao-a/go-outline@latest
GOBIN=/usr/local/bin/ go install go.uber.org/mock/mockgen@latest
#GOBIN=/usr/local/bin/ go install github.com/cortesi/modd/cmd/modd@latest
GOBIN=/usr/local/bin/ go install github.com/air-verse/air@latest