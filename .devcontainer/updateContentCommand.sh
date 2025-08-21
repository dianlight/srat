#!/usr/bin/env bash
set -x

apk add --no-cache git make lsblk eudev gcc musl-dev linux-headers samba ethtool e2fsprogs e2fsprogs-extra \
 fuse3 exfatprogs ntfs-3g-progs apfs-fuse openssh-client sshfs pre-commit shadow go \
 git-bash-completion git-prompt
apk add --no-cache --update-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community go "go~=1.25"
#bun
curl -fsSL https://bun.sh/install | bash -s "bun-v1.2.15"

make -C .. prepare || :

#gemini
bun add -g @google/gemini-cli ||:
bun pm -g trust --all ||:
sed -i '1s/node/bun/' "$(realpath $HOME/.bun/bin/gemini)" ||:

#biome
bun add -g biome ||:
bun pm -g trust --all ||:
sed -i '1s/node/bun/' "$(realpath $HOME/.bun/bin/biome)" ||:
