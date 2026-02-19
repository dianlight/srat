#!/usr/bin/env bash
set -x

apk add --no-cache git make lsblk eudev gcc musl-dev linux-headers samba ethtool e2fsprogs e2fsprogs-extra \
 fuse3 exfatprogs btrfs-progs dosfstools 	hfsprogs f2fs-tools xfsprogs-extra ntfs-3g-progs apfs-fuse openssh-client sshfs shadow \
 git-bash-completion git-prompt graphviz nix patch smartmontools zig minisign act \
 python3 py3-pip py3-mypy ruff py3-ruff py3-pytest

# Install Go from go.dev/dl/ – version extracted from backend/src/go.mod
GO_VERSION=$(grep '^go ' /workspaces/srat/backend/src/go.mod | awk '{print $2}')
GO_ARCH=$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')
echo "Installing Go ${GO_VERSION} for linux/${GO_ARCH}"
wget -nv "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz \
  && rm -rf /usr/local/go \
  && tar -C /usr/local -xzf /tmp/go.tar.gz \
  && rm /tmp/go.tar.gz \
  && ln -sf /usr/local/go/bin/go /usr/local/bin/go \
  && ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Documentation validation tools
apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community vale
apk add --no-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/testing lychee

#bun
BUN_VERSION=$(jq -r '.packageManager' /workspaces/srat/frontend/package.json | sed 's/bun@/bun-v/')
curl -fsSL https://bun.sh/install | bash -s "$BUN_VERSION"

make -C .. prepare || :

#gemini
#bun add -g @google/gemini-cli ||:
#bun pm -g trust --all ||:
#sed -i '1s/node/bun/' "$(realpath $HOME/.bun/bin/gemini)" ||:

# prek
bun install -g @j178/prek ||:

#biome
bun add -g biome @biomejs/cli-linux-arm64-musl ||:
bun pm -g trust --all ||:
#sed -i '1s/node/bun/' "$(realpath $HOME/.bun/bin/biome)" ||:
#export BIOME_BINARY=$(realpath $HOME/.bun/bin/biome)

#test device
mkdir -p /dev/disk/by-id
ln -s ../../vdb1 /dev/disk/by-id/1234-5678

#workarund for @rtk-query/codegen-openapi that don't work with bun
#apk add --no-cache nodejs npm
#npm install -g @rtk-query/codegen-openapi ||:

#enable nix
echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf ||:

#Use of act Workarount to have ver >=0.2.84
#gh extension install https://github.com/nektos/gh-act ||: 

#directory structure 
losetup -f /workspaces/srat/backend/test/data/image.dmg
for dir in /backup /config /ssl /addon_configs /addons /share /media ; do
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
        mount /dev/loop1 "$dir"
    fi
done
