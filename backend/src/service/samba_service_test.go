package service_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/dianlight/srat/dto"
	service "github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
	"github.com/ztrue/tracerr"
)

type SambaServiceSuite struct {
	suite.Suite
	sambaService service.SambaServiceInterface
	apictx       dto.ContextState
	//mockSambaService *MockSambaServiceInterface
	// VariableThatShouldStartAtFive int
}

func TestSambaServiceSuite(t *testing.T) {
	csuite := new(SambaServiceSuite)
	csuite.sambaService = service.NewSambaService(&csuite.apictx)
	var err error
	csuite.apictx.Template, err = os.ReadFile("../templates/smb.gtpl")
	csuite.apictx.DockerInterface = "hassio"
	csuite.apictx.DockerNet = "172.30.32.0/23"

	if err != nil {
		t.Errorf("Cant read template file %s", err)
	}
	/*
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		csuite.mockSambaService = NewMockSambaServiceInterface(ctrl)
		//csuite.mockBoradcaster.EXPECT().AddOpenConnectionListener(gomock.Any()).AnyTimes()
	*/
	suite.Run(t, csuite)
}

func (suite *SambaServiceSuite) TestCreateConfigStream() {
	stream, err := suite.sambaService.CreateConfigStream()
	suite.Require().NoError(err, tracerr.SprintSourceColor(err))
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
	suite.Len(keys, len(expected), result)
	m1 := regexp.MustCompile(`/\*(.*)\*/`)

	for k, v := range result {
		//assert.EqualValues(t, expected[k], v)
		var elines = strings.Split(expected[k], "\n")
		var lines = strings.Split(v, "\n")

		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "# DEBUG:") && strings.HasPrefix(strings.TrimSpace(elines[i]), "# DEBUG:") {
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

			suite.Require().EqualValues(strings.TrimSpace(elines[i]), strings.TrimSpace(line), "On Section [%s] Line:%d\n%d:\n%s\n%d:", k, i, low, strings.Join(lines[low:hight], "\n"), hight)
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
