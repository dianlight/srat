package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
)

// fakeReadFile builds a func(string)([]byte, error) that returns content for
// matching paths and ENOENT-equivalent errors for everything else.
func fakeReadFile(byPath map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		if data, ok := byPath[p]; ok {
			return []byte(data), nil
		}
		return nil, errors.New("file not found")
	}
}

//go:fix inline
func boolPtr(v bool) *bool { return new(v) }

func TestWholeDiskNameRe(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"sda", true},
		{"sdb", true},
		{"nvme0n1", true},
		{"nvme1n2", true},
		{"mmcblk0", true},
		{"mmcblk1", true},
		{"loop0", true},
		{"loop5", true},
		{"sda1", false},
		{"nvme0n1p1", false},
		{"mmcblk0p1", false},
		{"loop0p0", false},
		{"", false},
		{"sd", true},
		{"vda", true},
		{"sda1extra", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, wholeDiskNameRe.MatchString(tc.name))
		})
	}
}

func TestDetectRotational(t *testing.T) {
	tests := []struct {
		name             string
		devName          string
		sysFiles         map[string]string
		smartInfo        *dto.SmartInfo
		want             *bool
		expectNoReadFile bool
	}{
		{
			name:     "sysfs rotational=1 → HDD",
			devName:  "sda",
			sysFiles: map[string]string{"/sys/block/sda/queue/rotational": "1\n"},
			want:     new(true),
		},
		{
			name:     "sysfs rotational=0 → SSD",
			devName:  "sda",
			sysFiles: map[string]string{"/sys/block/sda/queue/rotational": "0\n"},
			want:     new(false),
		},
		{
			name:    "sysfs rotational=0 (no trailing newline) → SSD",
			devName: "nvme0n1",
			sysFiles: map[string]string{
				"/sys/block/nvme0n1/queue/rotational": "0",
			},
			want: new(false),
		},
		{
			name:      "sysfs missing, SMART supported with RPM>0 → HDD",
			devName:   "sda",
			sysFiles:  map[string]string{},
			smartInfo: &dto.SmartInfo{Supported: true, RotationRate: 7200},
			want:      new(true),
		},
		{
			name:      "sysfs missing, SMART supported with RPM=0 → SSD",
			devName:   "sda",
			sysFiles:  map[string]string{},
			smartInfo: &dto.SmartInfo{Supported: true, RotationRate: 0},
			want:      new(false),
		},
		{
			name:      "sysfs missing, SMART unsupported → unknown (nil)",
			devName:   "sda",
			sysFiles:  map[string]string{},
			smartInfo: &dto.SmartInfo{Supported: false, RotationRate: 0},
			want:      nil,
		},
		{
			name:      "sysfs missing, SMART nil → unknown (nil)",
			devName:   "sda",
			sysFiles:  map[string]string{},
			smartInfo: nil,
			want:      nil,
		},
		{
			name:    "sysfs garbled content → falls back to SMART",
			devName: "sda",
			sysFiles: map[string]string{
				"/sys/block/sda/queue/rotational": "yes",
			},
			smartInfo: &dto.SmartInfo{Supported: true, RotationRate: 5400},
			want:      new(true),
		},
		{
			name:      "empty devName → SMART fallback",
			devName:   "",
			smartInfo: &dto.SmartInfo{Supported: true, RotationRate: 7200},
			want:      new(true),
		},
		{
			// Sanitization must reject slashes before any filepath.Join call,
			// otherwise a malicious devName could read arbitrary files. Verify
			// by ensuring readFile is not invoked at all (smartInfo nil ⇒ nil).
			name:             "devName with slash (path traversal attempt) → SMART fallback",
			devName:          "../../../etc/passwd",
			sysFiles:         nil,
			smartInfo:        nil,
			want:             nil,
			expectNoReadFile: true,
		},
		{
			name:             "devName containing .. → SMART fallback",
			devName:          "sd..a",
			smartInfo:        nil,
			want:             nil,
			expectNoReadFile: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var readFileCalls int
			base := fakeReadFile(tc.sysFiles)
			spy := func(p string) ([]byte, error) {
				readFileCalls++
				return base(p)
			}
			h := &hardwareService{
				readFile:         spy,
				sysBlockBasePath: "/sys/block",
			}
			got := h.detectRotational(tc.devName, tc.smartInfo)
			if tc.expectNoReadFile {
				assert.Zero(t, readFileCalls, "readFile must not be called for traversal input")
			}
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.NotNil(t, got)
			assert.Equal(t, *tc.want, *got)
		})
	}
}

func TestDetectPartitionIncreases(t *testing.T) {
	nvmeTwoParts := partitionCountRecord{
		Serial: "SERIAL-A", DiskID: "nvme-a", LegacyName: "nvme0n1",
		Count:      2,
		Partitions: []string{"nvme0n1p1", "nvme0n1p2"},
	}
	nvmeThreeParts := partitionCountRecord{
		Serial: "SERIAL-A", DiskID: "nvme-a", LegacyName: "nvme0n1",
		Count:      3,
		Partitions: []string{"nvme0n1p1", "nvme0n1p2", "nvme0n1p3"},
	}
	nvmeOnePart := partitionCountRecord{
		Serial: "SERIAL-A", DiskID: "nvme-a", LegacyName: "nvme0n1",
		Count:      1,
		Partitions: []string{"nvme0n1p1"},
	}
	sdb := partitionCountRecord{
		Serial: "SERIAL-B", DiskID: "sdb", LegacyName: "sdb",
		Count:      1,
		Partitions: []string{"sdb1"},
	}

	tests := []struct {
		name        string
		prev        map[string]partitionCountRecord
		current     map[string]partitionCountRecord
		wantKeys    []string
		wantPrevCnt int
		wantCurCnt  int
		wantNewPart string
	}{
		{
			name:     "no previous snapshot → baseline, no anomaly",
			prev:     nil,
			current:  map[string]partitionCountRecord{"SERIAL-A": nvmeThreeParts},
			wantKeys: nil,
		},
		{
			name:     "stable count → no anomaly",
			prev:     map[string]partitionCountRecord{"SERIAL-A": nvmeTwoParts},
			current:  map[string]partitionCountRecord{"SERIAL-A": nvmeTwoParts},
			wantKeys: nil,
		},
		{
			name:        "increase 2→3 → flagged with new partition name",
			prev:        map[string]partitionCountRecord{"SERIAL-A": nvmeTwoParts},
			current:     map[string]partitionCountRecord{"SERIAL-A": nvmeThreeParts},
			wantKeys:    []string{"SERIAL-A"},
			wantPrevCnt: 2,
			wantCurCnt:  3,
			wantNewPart: "nvme0n1p3",
		},
		{
			name:     "decrease → not anomalous",
			prev:     map[string]partitionCountRecord{"SERIAL-A": nvmeThreeParts},
			current:  map[string]partitionCountRecord{"SERIAL-A": nvmeOnePart},
			wantKeys: nil,
		},
		{
			name:     "drive absent from previous snapshot → first sighting, no anomaly",
			prev:     map[string]partitionCountRecord{"SERIAL-B": sdb},
			current:  map[string]partitionCountRecord{"SERIAL-B": sdb, "SERIAL-A": nvmeThreeParts},
			wantKeys: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectPartitionIncreases(tc.prev, tc.current)
			if tc.wantKeys == nil {
				assert.Empty(t, got)
				return
			}
			assert.Len(t, got, len(tc.wantKeys))
			for _, key := range tc.wantKeys {
				inc, ok := got[key]
				assert.True(t, ok, "expected increase for key %s", key)
				assert.Equal(t, tc.wantPrevCnt, inc.Previous)
				assert.Equal(t, tc.wantCurCnt, inc.Current)
				assert.Equal(t, []string{tc.wantNewPart}, inc.NewPartition)
				assert.Equal(t, "SERIAL-A", inc.Serial)
				assert.Equal(t, "nvme-a", inc.DiskID)
			}
		})
	}
}

// TestRecordPartitionCounts_BaselineWarnAndRetention exercises the service-level
// recorder end-to-end: baseline publication must not warn, an increase must
// warn through slog with serial and new partition names, drives missing from a
// snapshot must keep their history, and decreases must update silently.
func TestRecordPartitionCounts_BaselineWarnAndRetention(t *testing.T) {
	var buf bytes.Buffer
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(oldDefault)

	h := &hardwareService{ctx: context.Background()}

	first := map[string]partitionCountRecord{
		"SERIAL-A": {
			Serial: "SERIAL-A", DiskID: "nvme-a", LegacyName: "nvme0n1",
			Count:      2,
			Partitions: []string{"nvme0n1p1", "nvme0n1p2"},
		},
		"SERIAL-B": {
			Serial: "SERIAL-B", DiskID: "sdb", LegacyName: "sdb",
			Count:      1,
			Partitions: []string{"sdb1"},
		},
	}
	h.recordPartitionCounts(first)

	firstLogs := buf.String()
	assert.NotContains(t, firstLogs, "Anomalous partition count increase",
		"first published snapshot is the diff baseline and must not warn")
	assert.Contains(t, firstLogs, "Hardware enumeration partition summary")
	assert.Contains(t, firstLogs, "serial=SERIAL-A")
	assert.Len(t, h.lastPartitionCounts, 2)

	buf.Reset()
	second := map[string]partitionCountRecord{
		"SERIAL-A": {
			Serial: "SERIAL-A", DiskID: "nvme-a", LegacyName: "nvme0n1",
			Count:      3,
			Partitions: []string{"nvme0n1p1", "nvme0n1p2", "nvme0n1p3"},
		},
		// SERIAL-B intentionally absent — its history must be retained.
	}
	h.recordPartitionCounts(second)

	warnLogs := buf.String()
	assert.Contains(t, warnLogs, "Anomalous partition count increase without local partitioning action")
	assert.Contains(t, warnLogs, "serial=SERIAL-A")
	assert.Contains(t, warnLogs, "disk_id=nvme-a")
	assert.Contains(t, warnLogs, "previous_partition_count=2")
	assert.Contains(t, warnLogs, "current_partition_count=3")
	assert.Contains(t, warnLogs, "new_partitions=nvme0n1p3")

	assert.Len(t, h.lastPartitionCounts, 2, "absent drive history must be retained across snapshots")
	assert.Equal(t, 3, h.lastPartitionCounts["SERIAL-A"].Count)
	assert.Equal(t, 1, h.lastPartitionCounts["SERIAL-B"].Count)

	buf.Reset()
	third := map[string]partitionCountRecord{
		"SERIAL-A": {
			Serial: "SERIAL-A", DiskID: "nvme-a", LegacyName: "nvme0n1",
			Count:      2,
			Partitions: []string{"nvme0n1p1", "nvme0n1p2"},
		},
	}
	h.recordPartitionCounts(third)

	assert.NotContains(t, buf.String(), "Anomalous partition count increase",
		"decreases are not anomalous and must not warn")
	assert.Equal(t, 2, h.lastPartitionCounts["SERIAL-A"].Count,
		"baseline must track the latest snapshot even on decrease")
}

// TestRecordPartitionCounts_NilMapReceiver verifies the lazy-init guard so the
// zero-value hardwareService used by focused unit tests cannot panic.
func TestRecordPartitionCounts_NilMapReceiver(t *testing.T) {
	h := &hardwareService{ctx: context.Background()}
	assert.Nil(t, h.lastPartitionCounts)
	h.recordPartitionCounts(map[string]partitionCountRecord{})
	assert.NotNil(t, h.lastPartitionCounts)
}
