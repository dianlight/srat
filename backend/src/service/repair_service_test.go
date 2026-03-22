package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/ctxkeys"
	"github.com/dianlight/srat/service"
	"github.com/stretchr/testify/suite"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type RepairServiceSuite struct {
	suite.Suite
	app    *fxtest.App
	repair service.RepairServiceInterface
}

func TestRepairServiceSuite(t *testing.T) {
	suite.Run(t, new(RepairServiceSuite))
}

func (suite *RepairServiceSuite) SetupTest() {
	suite.app = fxtest.New(suite.T(),
		fx.Provide(
			func() (context.Context, context.CancelFunc) {
				ctx := context.WithValue(context.Background(), ctxkeys.WaitGroup, &sync.WaitGroup{})
				return context.WithCancel(ctx)
			},
			func() *dto.ContextState { return &dto.ContextState{} },
			service.NewRepairService,
		),
		fx.Populate(&suite.repair),
	)

	suite.app.RequireStart()
}

func (suite *RepairServiceSuite) TearDownTest() {
	suite.app.RequireStop()
}

func (suite *RepairServiceSuite) TestCreateGetListAndDelete() {
	created, err := suite.repair.Create(dto.RepairCommandMessage{
		CommandID:      "cmd-1",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
		IsPersistent:   true,
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(created)
	suite.Equal(dto.RepairLifecycleStatusCreated, created.Status)

	record, ok := suite.repair.Get("disk_space_low")
	suite.True(ok)
	suite.Require().NotNil(record)
	suite.Equal("cmd-1", record.LastCommandID)

	records := suite.repair.List()
	suite.Len(records, 1)
	suite.Equal("disk_space_low", records[0].RepairID)

	err = suite.repair.Delete("disk_space_low")
	suite.Require().NoError(err)

	_, ok = suite.repair.Get("disk_space_low")
	suite.False(ok)
}

func (suite *RepairServiceSuite) TestUpdateAndApplyLifecycle() {
	_, err := suite.repair.Create(dto.RepairCommandMessage{
		CommandID:      "cmd-1",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().NoError(err)

	updated, err := suite.repair.Update(dto.RepairCommandMessage{
		CommandID:      "cmd-2",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionReconcile,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(updated)
	suite.Equal(dto.RepairLifecycleStatusUpdated, updated.Status)

	ignoreLifecycle, err := suite.repair.ApplyLifecycle(dto.RepairLifecycleMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		CommandID: "cmd-2",
		RepairID:  "disk_space_low",
		Status:    dto.RepairLifecycleStatusIgnored,
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(ignoreLifecycle)
	suite.Equal(dto.RepairLifecycleStatusIgnored, ignoreLifecycle.Status)

	errMessage := "failed in HA"
	errorLifecycle, err := suite.repair.ApplyLifecycle(dto.RepairLifecycleMessage{
		Type:      dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		CommandID: "cmd-2",
		RepairID:  "disk_space_low",
		Status:    dto.RepairLifecycleStatusError,
		Error:     &errMessage,
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(errorLifecycle)
	suite.Equal(dto.RepairLifecycleStatusError, errorLifecycle.Status)
	suite.Equal(&errMessage, errorLifecycle.LastError)
}

func (suite *RepairServiceSuite) TestCreateDuplicateAndMissingRecords() {
	_, err := suite.repair.Create(dto.RepairCommandMessage{
		CommandID:      "cmd-1",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().NoError(err)

	_, err = suite.repair.Create(dto.RepairCommandMessage{
		CommandID:      "cmd-2",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().Error(err)

	_, err = suite.repair.Update(dto.RepairCommandMessage{
		CommandID:      "cmd-3",
		RepairID:       "missing",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "missing",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().Error(err)

	err = suite.repair.Delete("missing")
	suite.Require().Error(err)

	_, err = suite.repair.ApplyLifecycle(dto.RepairLifecycleMessage{
		Type:     dto.ClientEventTypes.CLIENTEVENTTYPEREPAIRLIFECYCLE.String(),
		RepairID: "missing",
		Status:   dto.RepairLifecycleStatusDeleted,
	})
	suite.Require().Error(err)
}

func (suite *RepairServiceSuite) TestQueueAndFlushCommands() {
	suite.Equal(0, suite.repair.QueueSize())

	err := suite.repair.EnqueueCommand(dto.RepairCommandMessage{
		CommandID:      "cmd-q1",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().NoError(err)

	// Duplicate command_id should be ignored for idempotent replay behavior.
	err = suite.repair.EnqueueCommand(dto.RepairCommandMessage{
		CommandID:      "cmd-q1",
		RepairID:       "disk_space_low",
		Action:         dto.RepairCommandActionUpsert,
		TranslationKey: "disk_space_low",
		Severity:       dto.RepairIssueSeverityWarning,
	})
	suite.Require().NoError(err)
	suite.Equal(1, suite.repair.QueueSize())

	queued := suite.repair.FlushQueuedCommands()
	suite.Len(queued, 1)
	suite.Equal("cmd-q1", queued[0].CommandID)
	suite.Equal(0, suite.repair.QueueSize())

	queued = suite.repair.FlushQueuedCommands()
	suite.Nil(queued)
}
