package service_test

import (
	"context"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/dianlight/srat/dbom"
	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/homeassistant/mount"
	"github.com/dianlight/srat/repository"
	service "github.com/dianlight/srat/service"
	"github.com/ovechkin-dm/mockio/v2/matchers"
	"github.com/ovechkin-dm/mockio/v2/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type SambaServiceSuite struct {
	suite.Suite
	sambaService service.SambaServiceInterface
	//apictx              dto.ContextState
	exported_share_repo repository.ExportedShareRepositoryInterface
	property_repo       repository.PropertyRepositoryInterface
	samba_user_repo     repository.SambaUserRepositoryInterface
	ctrl                *matchers.MockController
	ctx                 context.Context
	cancel              context.CancelFunc
	app                 *fxtest.App
}

func TestSambaServiceSuite(t *testing.T) {
	suite.Run(t, new(SambaServiceSuite))
}

func (suite *SambaServiceSuite) SetupTest() {

	os.Setenv("HOSTNAME", "test-host")

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
			mock.Mock[service.DirtyDataServiceInterface],
			mock.Mock[service.SupervisorServiceInterface],
			mock.Mock[repository.ExportedShareRepositoryInterface],
			mock.Mock[repository.PropertyRepositoryInterface],
			mock.Mock[repository.SambaUserRepositoryInterface],
			mock.Mock[mount.ClientWithResponsesInterface],
		),
		fx.Populate(&suite.sambaService),
		fx.Populate(&suite.property_repo),
		fx.Populate(&suite.exported_share_repo),
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

func (suite *SambaServiceSuite) TestCreateConfigStream() {

	mock.When(suite.samba_user_repo.All()).ThenReturn(dbom.SambaUsers{
		{
			Username: "dianlight",
			IsAdmin:  true,
		},
	}, nil).Verify(matchers.Times(1))
	mock.When(suite.samba_user_repo.GetAdmin()).ThenReturn(dbom.SambaUser{}, nil).Verify(matchers.Times(0))
	mock.When(suite.samba_user_repo.GetUserByName("dianlight")).ThenReturn(&dbom.SambaUser{
		Username: "dianlight",
	}, nil).Verify(matchers.Times(0))

	mock.When(suite.property_repo.All(mock.Any[bool]())).ThenReturn(dbom.Properties{
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
			Value: true,
		},
		"CompatibilityMode": {
			Key:   "CompatibilityMode",
			Value: false,
		},
		"EnableRecycleBin": {
			Key:   "EnableRecycleBin",
			Value: false,
		},
		"VetoFiles": {
			Key:   "VetoFiles",
			Value: []string{"._*", ".DS_Store", "Thumbs.db", "icon?", ".Trashes"},
		},
	}, nil)

	mock.When(suite.exported_share_repo.All()).ThenReturn(&[]dbom.ExportedShare{
		{
			Name:               "CONFIG",
			MountPointDataPath: "/homeassistant",
			MountPointData: dbom.MountPointPath{
				Path: "/homeassistant",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
				},
			},
		},
		{
			Name:               "MEDIA",
			MountPointDataPath: "/media",
			MountPointData: dbom.MountPointPath{
				Path: "/media",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
		},
		{
			Name:               "BACKUP",
			MountPointDataPath: "/backup",
			MountPointData: dbom.MountPointPath{
				Path: "/backup",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
				},
			},
		},
		{
			Name:               "SHARE",
			MountPointDataPath: "/share",
			MountPointData: dbom.MountPointPath{
				Path: "/share",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
		},
		{
			Name:               "ADDONS",
			MountPointDataPath: "/addons",
			MountPointData: dbom.MountPointPath{
				Path: "/addons",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
		},
		{
			Name:               "ADDON_CONFIGS",
			MountPointDataPath: "/addon_configs",
			MountPointData: dbom.MountPointPath{
				Path: "/addon_configs",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
		},
		{
			Name:               "EFI",
			MountPointDataPath: "/mnt/EFI",
			MountPointData: dbom.MountPointPath{
				Path: "/mnt/EFI",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
		},
		{
			Name:               "LIBRARY",
			MountPointDataPath: "/mnt/LIBRARY",
			MountPointData: dbom.MountPointPath{
				Path: "/mnt/LIBRARY",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
		},
		{
			Name:               "UPDATER",
			MountPointDataPath: "/mnt/Updater",
			MountPointData: dbom.MountPointPath{
				Path: "/mnt/Updater",
			},
			Users: []dbom.SambaUser{
				{
					Username: "dianlight",
					IsAdmin:  true,
				},
			},
		},
	}, nil)

	stream, err := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(err)
	suite.NotNil(stream)

	fsbyte, err := os.ReadFile("../../test/data/smb.conf")
	suite.Require().NoError(err)

	var re = regexp.MustCompile(`(?m)^\[([^[]+)\]\n(?:^[^[].*\n+)+`)

	var result = make(map[string]string)
	//t.Log(fmt.Sprintf("%s", *stream))
	for _, match := range re.FindAllStringSubmatch(string(*stream), -1) {
		result[match[1]] = strings.TrimSpace(match[0])
	}

	var expected = make(map[string]string)
	for _, match := range re.FindAllStringSubmatch(string(fsbyte), -1) {
		expected[match[1]] = strings.TrimSpace(match[0])
	}

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	suite.Len(keys, len(expected), "%+v", result)
	m1 := regexp.MustCompile(`/\*(.*)\*/`)

	for k, v := range result {
		//assert.EqualValues(t, expected[k], v)
		var elines = strings.Split(expected[k], "\n")
		var lines = strings.Split(v, "\n")

		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "# ") && strings.HasPrefix(strings.TrimSpace(elines[i]), "# ") {
				continue
			}
			low := i - 5
			if low < 5 {
				low = 5
			}
			hight := low + 10
			if hight > len(lines) {
				hight = len(lines)
			}

			suite.Require().Greater(len(lines), i, "Premature End of file reached")
			if logv := m1.FindStringSubmatch(line); len(logv) > 1 {
				suite.T().Logf("%d: %s", i, logv[1])
				line = m1.ReplaceAllString(line, "")
			}

			suite.Require().Equal(strings.TrimSpace(elines[i]), strings.TrimSpace(line), "On Section [%s] Line:%d\n%d:\n%s\n%d:", k, i, low, strings.Join(lines[low:hight], "\n"), hight)
		}

	}
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
