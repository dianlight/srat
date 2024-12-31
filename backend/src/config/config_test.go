package config

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	InitDB(":memory:")
	m.Run()
}

func TestListMountPointDataEmpty(t *testing.T) {
	mountPoints, err := ListMountPointData()

	assert.NoError(t, err)
	assert.Equal(t, []MountPointData{}, mountPoints)
	assert.Len(t, mountPoints, 0)
}

func TestSaveMountPointData(t *testing.T) {

	testMountPoint := MountPointData{
		Path:   "/mnt/test",
		Label:  "Test Drive",
		Name:   "test_drive",
		FSType: "ext4",
		Flags:  []MounDataFlag{MS_RDONLY, MS_NOATIME},
		Data:   "rw,noatime",
	}

	err := SaveMountPointData(testMountPoint)

	assert.NoError(t, err)
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&MountPointData{})
}

func TestListMountPointData(t *testing.T) {

	expectedMountPoints := []MountPointData{
		{
			Path:   "/mnt/test1",
			Label:  "Test 1",
			Name:   "test1",
			FSType: "ext4",
			Flags:  []MounDataFlag{MS_RDONLY, MS_NOATIME},
			Data:   "rw,noatime",
		},
		{
			Path:   "/mnt/test2",
			Label:  "Test 2",
			Name:   "test2",
			FSType: "ntfs",
			Flags:  []MounDataFlag{MS_BIND},
			Data:   "bind",
		},
	}

	err := SaveMountPointData(expectedMountPoints[0])
	assert.NoError(t, err)
	err = SaveMountPointData(expectedMountPoints[1])
	assert.NoError(t, err)

	mountPoints, err := ListMountPointData()

	assert.NoError(t, err)
	if !cmp.Equal(expectedMountPoints, mountPoints, cmpopts.IgnoreFields(MountPointData{}, "CreatedAt", "UpdatedAt")) {
		assert.Equal(t, expectedMountPoints, mountPoints)
		//		t.Errorf("FuncUnderTest() mismatch")
	}
	//assert.Equal(t, expectedMountPoints, mountPoints)
	assert.Len(t, mountPoints, 2)

	for i, mp := range mountPoints {
		assert.Equal(t, expectedMountPoints[i].Path, mp.Path)
		assert.Equal(t, expectedMountPoints[i].Label, mp.Label)
		assert.Equal(t, expectedMountPoints[i].Name, mp.Name)
		assert.Equal(t, expectedMountPoints[i].FSType, mp.FSType)
		assert.Equal(t, expectedMountPoints[i].Flags, mp.Flags)
		assert.Equal(t, expectedMountPoints[i].Data, mp.Data)
	}
}

func TestSaveMountPointDataDuplicate(t *testing.T) {
	testMountPoint := MountPointData{
		Path:   "/mnt/test",
		Label:  "Test Drive",
		Name:   "test_drive",
		FSType: "ext4",
		Flags:  []MounDataFlag{MS_RDONLY, MS_NOATIME},
		Data:   "rw,noatime",
	}

	err := SaveMountPointData(testMountPoint)

	assert.NoError(t, err)
}

func TestSaveMountPointDataLargeNumber(t *testing.T) {
	numRecords := 1000
	testMountPoints := make([]MountPointData, numRecords)

	for i := 0; i < numRecords; i++ {
		testMountPoints[i] = MountPointData{
			Path:   fmt.Sprintf("/mnt/test%d", i),
			Label:  fmt.Sprintf("Test Drive %d", i),
			Name:   fmt.Sprintf("test_drive_%d", i),
			FSType: "ext4",
			Flags:  []MounDataFlag{MS_RDONLY, MS_NOATIME},
			Data:   "rw,noatime",
		}
	}

	for _, mp := range testMountPoints {
		err := SaveMountPointData(mp)
		assert.NoError(t, err)
	}
}
