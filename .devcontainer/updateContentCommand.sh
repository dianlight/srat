#!/bin/bash

apk add --no-cache go git make lsblk eudev gcc musl-dev linux-headers samba ethtool e2fsprogs e2fsprogs-extra fuse3 exfatprogs ntfs-3g-progs apfs-fuse

#bun
curl -fsSL https://bun.sh/install | bash

#Swag
#curl -s -L https://github.com/swaggo/swag/releases/download/v2.0.0-rc4/swag_2.0.0-rc4_Linux_arm64.tar.gz | tar xzvf - -C /usr/local/bin
#GOBIN=/usr/local/bin/ go install github.com/swaggo/swag/v2/cmd/swag@latest
GOBIN=/usr/local/bin/ go install github.com/rogpeppe/gohack@latest
GOBIN=/usr/local/bin/ go install github.com/rakyll/gotest@latest
GOBIN=/usr/local/bin/ go install github.com/Antonboom/testifylint@latest
GOBIN=/usr/local/bin/ go install github.com/ramya-rao-a/go-outline@latest
GOBIN=/usr/local/bin/ go install go.uber.org/mock/mockgen@latest