#!/usr/bin/env bash
set -x

#mise run //root:prepare || :

#test device
mkdir -p /dev/disk/by-id
ln -sf ../../vdb1 /dev/disk/by-id/1234-5678

#enable nix
echo "experimental-features = nix-command flakes" >>/etc/nix/nix.conf || :

#Use of act Workarount to have ver >=0.2.84
#gh extension install https://github.com/nektos/gh-act ||:

#directory structure
losetup -f /workspaces/srat/backend/test/data/image.dmg
for dir in /backup /config /ssl /addon_configs /addons /share /media; do
	if [ ! -d "$dir" ]; then
		mkdir -p "$dir"
		mount /dev/loop1 "$dir"
	fi
done
