package dbom

import (
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

/*
func TestMountPointDataAllEmpty(t *testing.T) {
	mountPoints, err := MountPointData{}.All()

	require.NoError(t, err)
	assert.Equal(t, []MountPointData{}, mountPoints)
	assert.Empty(t, mountPoints)
}
*/

func TestMountPointDataSaveWithoutData(t *testing.T) {

	testMountPoint := MountPointData{
		Path: "/addons",
	}

	err := testMountPoint.Save()

	require.NoError(t, err)
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&MountPointData{})
}

func TestMountPointDataSave(t *testing.T) {

	testMountPoint := MountPointData{
		Path: "/mnt/test",
		//Label:  "Test Drive",
		Source: "test_drive",
		FSType: "ext4",
		Flags:  []dto.MounDataFlag{dto.MS_RDONLY, dto.MS_NOATIME},
		//Data:   "rw,noatime",
		//DeviceId: 12344,
	}

	err := testMountPoint.Save()

	require.NoError(t, err)
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&MountPointData{})
}

func TestMountPointDataAll(t *testing.T) {

	expectedMountPoints := []MountPointData{
		{
			Path: "/mnt/test1",
			//Label:  "Test 1",
			Source: "test1",
			FSType: "ext4",
			Flags:  []dto.MounDataFlag{dto.MS_RDONLY, dto.MS_NOATIME},
			//Data:     "rw,noatime",
			DeviceId: 12345,
		},
		{
			Path: "/mnt/test2",
			//Label:  "Test 2",
			Source: "test2",
			FSType: "ntfs",
			Flags:  []dto.MounDataFlag{dto.MS_BIND},
			//Data:     "bind",
			DeviceId: 12346,
		},
	}

	err := expectedMountPoints[0].Save()
	require.NoError(t, err)
	err = expectedMountPoints[1].Save()
	require.NoError(t, err)

	mountPoints, err := MountPointData{}.All()

	require.NoError(t, err)
	if !cmp.Equal(expectedMountPoints, mountPoints, cmpopts.IgnoreFields(MountPointData{}, "CreatedAt", "UpdatedAt")) {
		assert.Equal(t, expectedMountPoints, mountPoints)
		//		t.Errorf("FuncUnderTest() mismatch")
	}
	//assert.Equal(t, expectedMountPoints, mountPoints)
	assert.Len(t, mountPoints, 2)

	for i, mp := range mountPoints {
		assert.Equal(t, expectedMountPoints[i].Path, mp.Path)
		//assert.Equal(t, expectedMountPoints[i].Label, mp.Label)
		assert.Equal(t, expectedMountPoints[i].Source, mp.Source)
		assert.Equal(t, expectedMountPoints[i].FSType, mp.FSType)
		assert.Equal(t, expectedMountPoints[i].Flags, mp.Flags)
		//assert.Equal(t, expectedMountPoints[i].Data, mp.Data)
	}
}

/*
func TestMountPointDataSaveDuplicate(t *testing.T) {
	testMountPoint := MountPointData{
		Path: "/mnt/test",
		//Label:  "Test Drive",
		Source: "test_drive",
		FSType: "ext4",
		Flags:  []dto.MounDataFlag{dto.MS_RDONLY, dto.MS_NOATIME},
		//Data:   "rw,noatime",
	}

	err := testMountPoint.Save()

	require.NoError(t, err)
}
*/

/*
	func TestMountPointDataSaveLargeNumber(t *testing.T) {
		numRecords := 1000
		testMountPoints := make([]MountPointData, numRecords)

		for i := 0; i < numRecords; i++ {
			testMountPoints[i] = MountPointData{
				Path: fmt.Sprintf("/mnt/test%d", i),
				//Label:  fmt.Sprintf("Test Drive %d", i),
				Name:   fmt.Sprintf("test_drive_%d", i),
				FSType: "ext4",
				Flags:  []dto.MounDataFlag{dto.MS_RDONLY, dto.MS_NOATIME},
				Data:   "rw,noatime",
			}
		}

		for _, mp := range testMountPoints {
			err := mp.Save()
			require.NoError(t, err)
		}
	}

	func TestMountPointDataSaveEmptyDefaultPath(t *testing.T) {
		testCases := []struct {
			name         string
			mountPoint   MountPointData
			expectedPath string
		}{
			{
				name: "Empty DefaultPath",
				mountPoint: MountPointData{
					Name:        "test_drive_23",
					Path:        "/mnt/test",
					DefaultPath: "",
				},
				expectedPath: "/mnt/test",
			},
			{
				name: "Non-empty DefaultPath",
				mountPoint: MountPointData{
					Name:        "test_drive_24",
					Path:        "/mnt/test",
					DefaultPath: "/mnt/original",
				},
				expectedPath: "/mnt/original",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.mountPoint.Save()
				require.NoError(t, err)
				assert.Equal(t, tc.expectedPath, tc.mountPoint.DefaultPath)
			})
		}
	}
*/
/*
func TestMountPointDataSaveWithSetDefaultPath(t *testing.T) {
	testMountPoint := MountPointData{
		Path: "/mnt/test",
	}

	err := testMountPoint.Save()

	require.NoError(t, err)
	assert.Equal(t, "/mnt/original", testMountPoint.DefaultPath)
}
*/
/*
func TestMountPointDataFromName(t *testing.T) {
	// Create a test mount point
	testMountPoint := MountPointData{
		Path: "/mnt/test",
		//Label:  "Test Drive",
		Source:   "test_drive",
		FSType:   "ext4",
		Flags:    []dto.MounDataFlag{dto.MS_RDONLY, dto.MS_NOATIME},
		Data:     "rw,noatime",
		DeviceId: 212345,
	}

	// Save the test mount point to the database
	err := testMountPoint.Save()
	require.NoError(t, err)

	// Create a new MountPointData instance to test FromName
	var retrievedMountPoint MountPointData

	// Call FromName with the existing name
	err = retrievedMountPoint.FromName("test_drive")
	t.Logf("%v", retrievedMountPoint)
	require.NoError(t, err)

	// Check if the retrieved mount point matches the original
	assert.Equal(t, testMountPoint.Path, retrievedMountPoint.Path)
	//assert.Equal(t, testMountPoint.Label, retrievedMountPoint.Label)
	assert.Equal(t, testMountPoint.Source, retrievedMountPoint.Source)
	assert.Equal(t, testMountPoint.FSType, retrievedMountPoint.FSType)
	assert.Equal(t, testMountPoint.Flags, retrievedMountPoint.Flags)
	assert.Equal(t, testMountPoint.Data, retrievedMountPoint.Data)

	// Clean up the database
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&MountPointData{})
}
*/
/*
func TestMountPointDataFromNameEmptyString(t *testing.T) {
	var mp MountPointData

	err := mp.FromName("")

	require.Error(t, err)
	require.ErrorContains(t, err, "name cannot be empty")
	assert.Empty(t, mp.Source)
	assert.Empty(t, mp.Path)
	//assert.Empty(t, mp.Label)
	assert.Empty(t, mp.FSType)
	assert.Empty(t, mp.Flags)
	assert.Empty(t, mp.Data)
}
*/
