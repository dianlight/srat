package service_test

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/dianlight/srat/converter"
	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/events"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/repository"
	service "github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SambaServiceSuite struct {
	suite.Suite
	sambaService service.SambaServiceInterface
	//apictx              dto.ContextState
	share_service   service.ShareServiceInterface
	property_repo   repository.PropertyRepositoryInterface
	samba_user_repo repository.SambaUserRepositoryInterface
	ctrl            *matchers.MockController
	ctx             context.Context
	cancel          context.CancelFunc
	app             *fxtest.App
}

func TestSambaServiceSuite(t *testing.T) {
	suite.Run(t, new(SambaServiceSuite))
}

func (suite *SambaServiceSuite) SetupTest() {
	data, err := os.ReadFile("../../test/data/mount_info.txt")
	if err != nil {
		log.Fatal(err)
	}
	osutil.MockMountInfo(string(data))

	fs := fstest.MapFS{
		"homeassistant": {
			Mode: os.ModeDir,
		},
		"media": {
			Mode: os.ModeDir,
		},
		"backup": {
			Mode: os.ModeDir,
		},
		"share": {
			Mode: os.ModeDir,
		},
		"addons": {
			Mode: os.ModeDir,
		},
		"addon_configs": {
			Mode: os.ModeDir,
		},
		"mnt/EFI": {
			Mode: os.ModeDir,
		},
		"mnt/LIBRARY": {
			Mode: os.ModeDir,
		},
		"mnt/Updater": {
			Mode: os.ModeDir,
		},
		"hello.txt": {
			Data: []byte("hello, world"),
		},
	}
	converter.MockFuncOsStat(fs.Stat)

	//os.Setenv("HOSTNAME", "test-host")

	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() *matchers.MockController { return mock.NewMockController(suite.T()) },
			func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.WithValue(context.Background(), "wg", &sync.WaitGroup{}))
			},
			func() *dto.ContextState {
				sharedResources := dto.ContextState{}
				sharedResources.DockerInterface = "hassio"
				sharedResources.DockerNet = "172.30.32.0/23"
				var err error
				sharedResources.Template, err = os.ReadFile("../templates/smb.gtpl")
				if err != nil {
					suite.T().Errorf("Cant read template file %s", err)
				}

				return &sharedResources
			},
			service.NewSambaService,
			service.NewShareService,
			mock.Mock[service.BroadcasterServiceInterface],
			mock.Mock[service.DirtyDataServiceInterface],
			mock.Mock[service.SupervisorServiceInterface],
			mock.Mock[repository.ExportedShareRepositoryInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[repository.SambaUserRepositoryInterface],
			mock.Mock[repository.MountPointPathRepositoryInterface],
			mock.Mock[mount.ClientWithResponsesInterface],
			mock.Mock[service.HaWsServiceInterface],
			mock.Mock[service.HDIdleServiceInterface],
			mock.Mock[events.EventBusInterface],
		),
		fx.Populate(&suite.sambaService),
		fx.Populate(&suite.property_repo),
		fx.Populate(&suite.share_service),
		fx.Populate(&suite.samba_user_repo),
		fx.Populate(&suite.ctx),
		fx.Populate(&suite.cancel),
	)
	suite.app.RequireStart()
}

func (suite *SambaServiceSuite) TearDownTest() {
	suite.cancel()
	suite.ctx.Value("wg").(*sync.WaitGroup).Wait()
	suite.app.RequireStop()
}

// Helper function to setup common test data
func (suite *SambaServiceSuite) setupCommonMocks() {
	mock.When(suite.samba_user_repo.All()).ThenReturn(dbom.SambaUsers{
		{
			Username: "dianlight",
			IsAdmin:  true,
		},
		{
			Username: "testuser",
			IsAdmin:  false,
		},
	}, nil)

	mock.When(suite.property_repo.All(mock.Any[bool]())).ThenReturn(dbom.Properties{
		"Hostname": {
			Key:   "Hostname",
			Value: "test-host",
		},
		"Workgroup": {
			Key:   "Workgroup",
			Value: "WORKGROUP",
		},
		"AllowHost": {
			Key:   "AllowHost",
			Value: []string{"10.0.0.0/8", "100.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "fe80::/10", "fc00::/7"},
		},
		"Interfaces": {
			Key:   "Interfaces",
			Value: []string{"wlan0", "end0"},
		},
		"BindAllInterfaces": {
			Key:   "BindAllInterfaces",
			Value: false,
		},
		"CompatibilityMode": {
			Key:   "CompatibilityMode",
			Value: false,
		},
		"EnableRecycleBin": {
			Key:   "EnableRecycleBin",
			Value: false,
		},
	}, nil)

	mock.When(suite.share_service.All()).ThenReturn(&[]dbom.ExportedShare{
		{
			Name:               "CONFIG",
			MountPointDataPath: "/homeassistant",
			MountPointData: dbom.MountPointPath{
				Path: "homeassistant",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
				},
			},
			VetoFiles: []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
		{
			Name:               "MEDIA",
			MountPointDataPath: "/media",
			MountPointData: dbom.MountPointPath{
				Path: "media",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
			VetoFiles: []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
		{
			Name:               "BACKUP",
			MountPointDataPath: "/backup",
			MountPointData: dbom.MountPointPath{
				Path: "backup",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
				},
			},
			VetoFiles: []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
		{
			Name:               "SHARE",
			MountPointDataPath: "/share",
			MountPointData: dbom.MountPointPath{
				Path: "share",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
			VetoFiles: []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
		{
			Name:               "ADDONS",
			MountPointDataPath: "/addons",
			MountPointData: dbom.MountPointPath{
				Path: "addons",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
			VetoFiles: []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
		{
			Name:               "ADDON_CONFIGS",
			MountPointDataPath: "/addon_configs",
			MountPointData: dbom.MountPointPath{
				Path: "addon_configs",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
			VetoFiles: []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
		{
			Name:               "EFI",
			MountPointDataPath: "/mnt/EFI",
			MountPointData: dbom.MountPointPath{
				Path: "mnt/EFI",
			},
			Users: []dbom.SambaUser{
				{
					Username: "testuser",
					IsAdmin:  false,
				},
			},
			RoUsers: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
			VetoFiles: []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
		{
			Name:               "LIBRARY",
			MountPointDataPath: "/mnt/LIBRARY",
			MountPointData: dbom.MountPointPath{
				Path: "mnt/LIBRARY",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
			TimeMachine: true,
			VetoFiles:   []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
		{
			Name:               "UPDATER",
			MountPointDataPath: "/mnt/Updater",
			MountPointData: dbom.MountPointPath{
				Path: "mnt/Updater",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
			RecycleBin: true,
			VetoFiles:  []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
	}, nil)
}

// Helper function to compare generated config against expected template
func (suite *SambaServiceSuite) compareConfigSections(generatedConfig *[]byte, testName string, expectedSections map[string]string) {
	dmp := diffmatchpatch.New()
	var re = regexp.MustCompile(`(?m)^\[([^[]+)\]\n(?:^[^[].*\n+)+`)

	var result = make(map[string]string)
	for _, match := range re.FindAllStringSubmatch(string(*generatedConfig), -1) {
		result[match[1]] = strings.TrimSpace(match[0])
	}

	suite.Len(result, len(expectedSections), "Test: %s - Expected %d sections, got %d", testName, len(expectedSections), len(result))

	for sectionName, expectedSection := range expectedSections {
		actualSection, ok := result[sectionName]
		suite.True(ok, "Test: %s - Section [%s] missing from generated config", testName, sectionName)
		if !ok {
			continue
		}

		var elines = strings.Split(strings.TrimSpace(expectedSection), "\n")
		var lines = strings.Split(strings.TrimSpace(actualSection), "\n")

		filterComments := func(inputLines []string) []string {
			var filtered []string
			for _, line := range inputLines {
				trimmedLine := strings.TrimSpace(line)
				if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
					filtered = append(filtered, trimmedLine)
				}
			}
			return filtered
		}

		filteredExpectedLines := filterComments(elines)
		filteredActualLines := filterComments(lines)

		filteredExpectedString := strings.Join(filteredExpectedLines, "\n")
		filteredActualString := strings.Join(filteredActualLines, "\n")

		rawDiffs := dmp.DiffMain(filteredExpectedString, filteredActualString, false)

		var foundDifference bool
		if len(rawDiffs) > 0 {
			for _, d := range rawDiffs {
				if d.Type == diffmatchpatch.DiffInsert || d.Type == diffmatchpatch.DiffDelete {
					foundDifference = true
					break
				}
			}
		}

		if foundDifference {
			semanticDiffs := dmp.DiffCleanupSemantic(rawDiffs)
			var diffEvidenceBuilder strings.Builder
			for _, d := range semanticDiffs {
				scanner := bufio.NewScanner(strings.NewReader(d.Text))
				for scanner.Scan() {
					line := scanner.Text()
					switch d.Type {
					case diffmatchpatch.DiffDelete:
						diffEvidenceBuilder.WriteString(fmt.Sprintf("- %s\n", line))
					case diffmatchpatch.DiffInsert:
						diffEvidenceBuilder.WriteString(fmt.Sprintf("+ %s\n", line))
					case diffmatchpatch.DiffEqual:
						diffEvidenceBuilder.WriteString(fmt.Sprintf("  %s\n", line))
					}
				}
			}
			suite.T().Errorf("Test: %s - Config mismatch for section [%s]:\n%s", testName, sectionName, diffEvidenceBuilder.String())
		}
	}
}

// Base test with Samba 4.23 (latest modern version)
func (suite *SambaServiceSuite) TestCreateConfigStream() {
	defer osutil.MockSambaVersion("4.23.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	suite.NotNil(stream)

	fsbyte, err := os.ReadFile("../../test/data/smb.conf")
	suite.Require().NoError(err)

	var re = regexp.MustCompile(`(?m)^\[([^[]+)\]\n(?:^[^[].*\n+)+`)

	var expected = make(map[string]string)
	for _, match := range re.FindAllStringSubmatch(string(fsbyte), -1) {
		expected[match[1]] = strings.TrimSpace(match[0])
	}

	suite.compareConfigSections(stream, "Samba4.23", expected)
}

// Test with Samba 4.21.0 - earliest supported version
func (suite *SambaServiceSuite) TestCreateConfigStream_Samba421() {
	defer osutil.MockSambaVersion("4.21.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	suite.NotNil(stream)

	// Samba 4.21 expectations:
	// - Should include fruit:posix_rename (4.22 removed it)
	// - Should NOT include server smb transports (4.23+ feature)
	configStr := string(*stream)
	suite.Contains(configStr, "fruit:posix_rename", "Samba 4.21 should include fruit:posix_rename")
	suite.NotContains(configStr, "server smb transports", "Samba 4.21 should NOT include server smb transports")
}

// Test with Samba 4.22.0 - middle supported version
func (suite *SambaServiceSuite) TestCreateConfigStream_Samba422() {
	defer osutil.MockSambaVersion("4.22.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	suite.NotNil(stream)

	// Samba 4.22 expectations:
	// - Should NOT include fruit:posix_rename (removed in 4.22)
	// - Should NOT include server smb transports (4.23+ feature)
	configStr := string(*stream)
	suite.NotContains(configStr, "fruit:posix_rename = yes", "Samba 4.22 should NOT include fruit:posix_rename")
	suite.NotContains(configStr, "server smb transports", "Samba 4.22 should NOT include server smb transports")
}

// Test with Samba 4.23.0 - latest modern version with full features
func (suite *SambaServiceSuite) TestCreateConfigStream_Samba423() {
	defer osutil.MockSambaVersion("4.23.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	suite.NotNil(stream)

	// Samba 4.23 expectations:
	// - Should NOT include fruit:posix_rename (removed in 4.22)
	// - Should include server smb transports (4.23+ feature)
	configStr := string(*stream)
	suite.NotContains(configStr, "fruit:posix_rename = yes", "Samba 4.23 should NOT include fruit:posix_rename")
	suite.Contains(configStr, "server smb transports", "Samba 4.23 should include server smb transports")
}

// Test with Samba 4.24.0 - future version
func (suite *SambaServiceSuite) TestCreateConfigStream_Samba424() {
	defer osutil.MockSambaVersion("4.24.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	suite.NotNil(stream)

	// Samba 4.24 expectations (same as 4.23+):
	// - Should NOT include fruit:posix_rename
	// - Should include server smb transports
	configStr := string(*stream)
	suite.NotContains(configStr, "fruit:posix_rename = yes", "Samba 4.24 should NOT include fruit:posix_rename")
	suite.Contains(configStr, "server smb transports", "Samba 4.24 should include server smb transports")
}

// Test with Samba 5.0.0 - major version bump (hypothetical future)
func (suite *SambaServiceSuite) TestCreateConfigStream_Samba500() {
	defer osutil.MockSambaVersion("5.0.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	suite.NotNil(stream)

	// Samba 5.0 should maintain forward compatibility
	configStr := string(*stream)
	suite.NotContains(configStr, "fruit:posix_rename = yes", "Samba 5.0 should NOT include fruit:posix_rename")
	suite.Contains(configStr, "server smb transports", "Samba 5.0 should include server smb transports")
}

// Test with unparseable version string (should fallback gracefully)
func (suite *SambaServiceSuite) TestCreateConfigStream_InvalidVersion() {
	defer osutil.MockSambaVersion("invalid-version-string")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	// Should still succeed but with safe defaults
	suite.Require().NoError(errE)
	suite.NotNil(stream)

	// With invalid version, should fallback to conservative defaults
	configStr := string(*stream)
	// Should not crash and produce valid config output
	suite.Contains(configStr, "[global]", "Config should still have global section")
}

// Test with empty version string (edge case)
func (suite *SambaServiceSuite) TestCreateConfigStream_EmptyVersion() {
	defer osutil.MockSambaVersion("")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	suite.NotNil(stream)

	configStr := string(*stream)
	suite.Contains(configStr, "[global]", "Config should still have global section even with empty version")
}

// Test version comparison logic for boundary conditions
func (suite *SambaServiceSuite) TestCreateConfigStream_VersionBoundary_4_21_9() {
	// Test version 4.21.9 (should still be 4.21 behavior)
	defer osutil.MockSambaVersion("4.21.9")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)

	configStr := string(*stream)
	suite.Contains(configStr, "fruit:posix_rename", "Samba 4.21.9 should include fruit:posix_rename")
}

// Test version comparison logic for boundary conditions
func (suite *SambaServiceSuite) TestCreateConfigStream_VersionBoundary_4_22_1() {
	// Test version 4.22.1 (should be 4.22 behavior)
	defer osutil.MockSambaVersion("4.22.1")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)

	configStr := string(*stream)
	suite.NotContains(configStr, "fruit:posix_rename = yes", "Samba 4.22.1 should NOT include fruit:posix_rename")
}

// Test version comparison logic for boundary conditions
func (suite *SambaServiceSuite) TestCreateConfigStream_VersionBoundary_4_23_0() {
	// Test version 4.23.0 (exact match)
	defer osutil.MockSambaVersion("4.23.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)

	configStr := string(*stream)
	suite.Contains(configStr, "server smb transports", "Samba 4.23.0 should include server smb transports")
}

// Test version with patch level variations
// These tests verify that version boundaries are correctly detected
func (suite *SambaServiceSuite) TestCreateConfigStream_VersionPatchVariations_4_20() {
	defer osutil.MockSambaVersion("4.20.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	configStr := string(*stream)

	// Samba 4.20 (before 4.21) - should have fruit:posix_rename
	suite.Contains(configStr, "fruit:posix_rename", "Samba 4.20 should include fruit:posix_rename")
	suite.NotContains(configStr, "server smb transports =", "Samba 4.20 should NOT include server smb transports")
}

func (suite *SambaServiceSuite) TestCreateConfigStream_VersionPatchVariations_4_21_17() {
	defer osutil.MockSambaVersion("4.21.17")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	configStr := string(*stream)

	// Samba 4.21.17 - should have fruit:posix_rename
	suite.Contains(configStr, "fruit:posix_rename", "Samba 4.21.17 should include fruit:posix_rename")
	suite.NotContains(configStr, "server smb transports =", "Samba 4.21.17 should NOT include server smb transports")
}

func (suite *SambaServiceSuite) TestCreateConfigStream_VersionPatchVariations_4_22_10() {
	defer osutil.MockSambaVersion("4.22.10")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	configStr := string(*stream)

	// Samba 4.22.10 - should NOT have fruit:posix_rename
	suite.NotContains(configStr, "fruit:posix_rename = yes", "Samba 4.22.10 should NOT include fruit:posix_rename")
	suite.NotContains(configStr, "server smb transports =", "Samba 4.22.10 should NOT include server smb transports")
}

func (suite *SambaServiceSuite) TestCreateConfigStream_VersionPatchVariations_4_23_5() {
	defer osutil.MockSambaVersion("4.23.5")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	configStr := string(*stream)

	// Samba 4.23.5 - should NOT have fruit:posix_rename but SHOULD have server smb transports
	suite.NotContains(configStr, "fruit:posix_rename = yes", "Samba 4.23.5 should NOT include fruit:posix_rename")
	suite.Contains(configStr, "server smb transports", "Samba 4.23.5 should include server smb transports")
}

func (suite *SambaServiceSuite) TestCreateConfigStream_VersionPatchVariations_4_24_0() {
	defer osutil.MockSambaVersion("4.24.0")()
	suite.setupCommonMocks()

	stream, errE := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(errE)
	configStr := string(*stream)

	// Samba 4.24.0 - forward compatible with 4.23+
	suite.NotContains(configStr, "fruit:posix_rename = yes", "Samba 4.24.0 should NOT include fruit:posix_rename")
	suite.Contains(configStr, "server smb transports", "Samba 4.24.0 should include server smb transports")
}

/*
func (suite *SambaHandlerSuite) checkStringInSMBConfig(testvalue string, expected string, t *testing.T) bool {
	stream, err := suite.CreateConfigStream(testContext)
	require.NoError(t, err)
	assert.NotNil(t, stream)

	rexpt := fmt.Sprintf(expected, testvalue)

	m, err := regexp.MatchString(rexpt, string(*stream))
	require.NoError(t, err)
	assert.True(t, m, "Wrong Match `%s` not found in stream \n%s", rexpt, string(*stream))

	return true
}
*/
