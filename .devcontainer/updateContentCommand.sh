#!/usr/bin/env bash
set -x

apk add --no-cache git make lsblk eudev gcc musl-dev linux-headers samba ethtool e2fsprogs e2fsprogs-extra \
 fuse3 exfatprogs ntfs-3g-progs apfs-fuse openssh-client sshfs pre-commit shadow go \
 git-bash-completion git-prompt graphviz nix patch smartmontools zig act cargo
apk add --no-cache --update-cache --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community go "go~=1.25"

#bun
curl -fsSL https://bun.sh/install | bash -s "bun-v1.3.5"

# Lychee - Link checker for documentation validation
cargo install lychee --locked || echo "Lychee installation failed, skipping"

# Vale - Prose linter for documentation validation
VALE_VERSION="3.10.0"
wget "https://github.com/errata-ai/vale/releases/download/v${VALE_VERSION}/vale_${VALE_VERSION}_Linux_64-bit.tar.gz" -O /tmp/vale.tar.gz && \
  tar -xzf /tmp/vale.tar.gz -C /tmp && \
  mv /tmp/vale /usr/local/bin/vale && \
  chmod +x /usr/local/bin/vale && \
  rm /tmp/vale.tar.gz || echo "Vale installation failed, skipping"

make -C .. prepare || :

#gemini
#bun add -g @google/gemini-cli ||:
#bun pm -g trust --all ||:
#sed -i '1s/node/bun/' "$(realpath $HOME/.bun/bin/gemini)" ||:

#biome
bun add -g biome ||:
bun pm -g trust --all ||:
sed -i '1s/node/bun/' "$(realpath $HOME/.bun/bin/biome)" ||:

#test device
mkdir -p /dev/disk/by-id
ln -s ../../vdb1 /dev/disk/by-id/1234-5678

#workarund for @rtk-query/codegen-openapi that don't work with bun
#apk add --no-cache nodejs npm
#npm install -g @rtk-query/codegen-openapi ||:

#enable nix
echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf ||:

#Use of act
#gh extension install https://github.com/nektos/gh-act ||:

#directory structure 
losetup -f /workspaces/srat/backend/test/data/image.dmg
for dir in /backup /config /ssl /addon_configs /addons /share /media ; do
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
        mount /dev/loop1 "$dir"
    fi
done
