#!/bin/bash
FS="$1"
DEV="$2"
LABEL="$3"
BASE="http://127.0.0.1:62983"
SID="fs$(date +%s)"
SHARE="fstest_${FS}"
MOUNTPATH="/mnt/${LABEL}"
echo "===== TESTING FS: $FS on $DEV ($LABEL) ====="

# 0. cleanup prior
curl -s -X DELETE "$BASE/api/share/$SHARE" >/dev/null 2>&1
curl -s -X DELETE "$BASE/api/volume?mount_path=$(printf '%s' "$MOUNTPATH" | sed 's#/#%2F#g')&force=true" >/dev/null 2>&1
sleep 1

# 1. Format
echo "--- FORMAT ---"
FR=$(curl -s -X POST "$BASE/api/filesystem/format" -H "Content-Type: application/json" -d "{\"partitionId\":\"$DEV\",\"filesystemType\":\"$FS\",\"label\":\"$LABEL\",\"force\":true,\"verbose\":false}")
echo "$FR" | jq -c '{success,message,exitCode}' 2>/dev/null || echo "$FR"
sleep 4
FT=$(curl -s "$BASE/api/volumes" | jq -r "[.[] | .partitions | to_entries[] | select(.key==\"$DEV\") | .value.fs_type][0]")
echo "fs_type after format: $FT"
if [ "$FT" != "$FS" ]; then
	echo "FORMAT FAILED ($FT != $FS)"
	exit 1
fi

# 2. Mount
echo "--- MOUNT ---"
MR=$(curl -s -X POST "$BASE/api/volume/mount" -H "Content-Type: application/json" -d "{\"path\":\"$MOUNTPATH\",\"root\":\"/\",\"fstype\":\"$FS\",\"device_id\":\"$DEV\",\"type\":\"ADDON\",\"is_mounted\":false,\"is_to_mount_at_startup\":false,\"flags\":[],\"custom_flags\":[]}")
echo "$MR" | jq -c '{path,is_mounted,fstype}' 2>/dev/null
if ! echo "$MR" | jq -e '.is_mounted==true' >/dev/null 2>&1; then
	echo "MOUNT FAILED"
	echo "$MR"
	exit 1
fi
MPD=$(echo "$MR" | jq -c '.')

# 3. Create share
echo "--- CREATE SHARE ---"
SR=$(curl -s -X POST "$BASE/api/share" -H "Content-Type: application/json" -d "{\"name\":\"$SHARE\",\"users\":[{\"username\":\"admin\",\"is_admin\":true},{\"username\":\"fsuser1\"},{\"username\":\"fsuser2\"}],\"ro_users\":[{\"username\":\"fsuser3\"}],\"guest_ok\":false,\"usage\":\"share\",\"mount_point_data\":$MPD}")
echo "$SR" | jq -c '{name,status,users:[.users[].username],ro:[.ro_users[].username]}' 2>/dev/null
if ! echo "$SR" | jq -e '.name' >/dev/null 2>&1; then
	echo "SHARE CREATE FAILED"
	echo "$SR"
	exit 1
fi
sleep 3

# 4. Samba list
echo "--- SAMBA LIST ---"
docker exec app_local_sambanas2 smbclient -L //127.0.0.1 -U admin%overnice 2>&1 | grep -i "$SHARE" || echo "SHARE NOT IN SAMBA LIST"

# 5. tests
UNAME=$(echo "$SHARE" | tr '[:lower:]' '[:upper:]')
echo "--- GUEST MOUNT ---"
mkdir -p /tmp/scratchpad/mntguest_"${SID}"
mount.cifs //192.168.0.68/"${UNAME}" /tmp/scratchpad/mntguest_"${SID}" -o guest,vers=3.0 2>&1 | head -1
echo "guest (expect Permission denied): $?"
umount /tmp/scratchpad/mntguest_"${SID}" 2>/dev/null

echo "--- ADMIN RW ---"
docker exec app_local_sambanas2 smbclient //127.0.0.1/"${UNAME}" -U admin%overnice -c "mkdir admin-${SID}; ls admin-${SID}" 2>&1 | tail -2

echo "--- USER1 RW ---"
docker exec app_local_sambanas2 smbclient //127.0.0.1/"${UNAME}" -U fsuser1%changeme -c "mkdir user1-${SID}; cd admin-${SID}; mkdir inadmin; ls" 2>&1 | tail -4

echo "--- USER2 RW ---"
docker exec app_local_sambanas2 smbclient //127.0.0.1/"${UNAME}" -U fsuser2%changeme -c "cd admin-${SID}; mkdir u2dir; cd ../user1-${SID}; mkdir u2dir2" 2>&1 | tail -2

echo "--- USER3 RO ---"
docker exec app_local_sambanas2 smbclient //127.0.0.1/"${UNAME}" -U fsuser3%changeme -c "mkdir shouldfail-${SID}" 2>&1 | tail -2

# 6. Cleanup test dirs
echo "--- CLEANUP ---"
docker exec app_local_sambanas2 smbclient //127.0.0.1/"${UNAME}" -U admin%overnice -c "cd admin-${SID}; rmdir inadmin; cd ..; rmdir admin-${SID}; rmdir user1-${SID}" 2>&1 | tail -1
curl -s -X DELETE "$BASE/api/share/$SHARE" >/dev/null 2>&1
curl -s -X DELETE "$BASE/api/volume?mount_path=$(printf '%s' "$MOUNTPATH" | sed 's#/#%2F#g')&force=true" >/dev/null 2>&1
echo "===== DONE $FS ====="
