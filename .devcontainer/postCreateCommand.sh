#!/bin/bash

apk add  --no-cache go git make lsblk eudev gcc musl-dev linux-headers samba

#bun
curl -fsSL https://bun.sh/install | bash 

#Swag
#curl -s -L https://github.com/swaggo/swag/releases/download/v2.0.0-rc4/swag_2.0.0-rc4_Linux_arm64.tar.gz | tar xzvf - -C /usr/local/bin
GOBIN=/usr/local/bin/ go install github.com/swaggo/swag/v2/cmd/swag@latest
