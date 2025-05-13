#!/bin/bash

apk add --no-cache git make lsblk eudev gcc musl-dev linux-headers samba ethtool e2fsprogs e2fsprogs-extra fuse3 exfatprogs ntfs-3g-progs apfs-fuse openssh-client sshfs
apk add --no-cache --update-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community  go "go=1.24.2-r1" 
#bun
curl -fsSL https://bun.sh/install | bash -s "bun-v1.2.10" 

#GOBIN=/usr/local/bin/ go install github.com/rogpeppe/gohack@v1.0.2
#GOBIN=/usr/local/bin/ go install github.com/rakyll/gotest@v0.0.6
#GOBIN=/usr/local/bin/ go install github.com/Antonboom/testifylint@v1.6.0
##GOBIN=/usr/local/bin/ go install github.com/ramya-rao-a/go-outline@1.0.0
##GOBIN=/usr/local/bin/ go install go.uber.org/mock/mockgen@v0.5.0
##GOBIN=/usr/local/bin/ go install github.com/cortesi/modd/cmd/modd@latest
#GOBIN=/usr/local/bin/ go install github.com/air-verse/air@v1.61.7
