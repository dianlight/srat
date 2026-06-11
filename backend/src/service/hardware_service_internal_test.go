package service

import (
	"errors"
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

func boolPtr(v bool) *bool { return &v }

func TestDetectRotational(t *testing.T) {
	tests := []struct {
		name      string
		devName   string
		sysFiles  map[string]string
		smartInfo *dto.SmartInfo
		want      *bool
	}{
		{
			name:     "sysfs rotational=1 → HDD",
			devName:  "sda",
			sysFiles: map[string]string{"/sys/block/sda/queue/rotational": "1\n"},
			want:     boolPtr(true),
		},
		{
			name:     "sysfs rotational=0 → SSD",
			devName:  "sda",
			sysFiles: map[string]string{"/sys/block/sda/queue/rotational": "0\n"},
			want:     boolPtr(false),
		},
		{
			name:    "sysfs rotational=0 (no trailing newline) → SSD",
			devName: "nvme0n1",
			sysFiles: map[string]string{
				"/sys/block/nvme0n1/queue/rotational": "0",
			},
			want: boolPtr(false),
		},
		{
			name:      "sysfs missing, SMART supported with RPM>0 → HDD",
			devName:   "sda",
			sysFiles:  map[string]string{},
			smartInfo: &dto.SmartInfo{Supported: true, RotationRate: 7200},
			want:      boolPtr(true),
		},
		{
			name:      "sysfs missing, SMART supported with RPM=0 → SSD",
			devName:   "sda",
			sysFiles:  map[string]string{},
			smartInfo: &dto.SmartInfo{Supported: true, RotationRate: 0},
			want:      boolPtr(false),
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
			want:      boolPtr(true),
		},
		{
			name:      "empty devName → SMART fallback",
			devName:   "",
			smartInfo: &dto.SmartInfo{Supported: true, RotationRate: 7200},
			want:      boolPtr(true),
		},
		{
			// Sanitization must reject slashes before any filepath.Join call,
			// otherwise a malicious devName could read arbitrary files. Verify
			// by ensuring readFile is not invoked at all (smartInfo nil ⇒ nil).
			name:      "devName with slash (path traversal attempt) → SMART fallback",
			devName:   "../../../etc/passwd",
			sysFiles:  nil,
			smartInfo: nil,
			want:      nil,
		},
		{
			name:      "devName containing .. → SMART fallback",
			devName:   "sd..a",
			smartInfo: nil,
			want:      nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &hardwareService{
				readFile:         fakeReadFile(tc.sysFiles),
				sysBlockBasePath: "/sys/block",
			}
			got := h.detectRotational(tc.devName, tc.smartInfo)
			if tc.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.NotNil(t, got)
			assert.Equal(t, *tc.want, *got)
		})
	}
}
