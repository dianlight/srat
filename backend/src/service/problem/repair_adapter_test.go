package problem

import (
	"context"
	"testing"

	"github.com/dianlight/srat/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	upsertErr    error
	dismissErr   error
	lifecycleErr error
	lastProblem  *dto.Problem
	lastKey      string
	lastStatus   dto.ProblemLifecycleStatus
	upserts      int
	dismisses    int
	lifecycles   int
}

func (f *fakeStore) Upsert(problem *dto.Problem) (*dto.Problem, error) {
	f.upserts++
	f.lastProblem = problem
	if f.upsertErr != nil {
		return nil, f.upsertErr
	}
	return problem, nil
}

func (f *fakeStore) Dismiss(problemKey string) error {
	f.dismisses++
	f.lastKey = problemKey
	return f.dismissErr
}

func (f *fakeStore) ApplyLifecycle(problemKey string, status dto.ProblemLifecycleStatus, lastError *string) (*dto.Problem, error) {
	f.lifecycles++
	f.lastKey = problemKey
	f.lastStatus = status
	if f.lifecycleErr != nil {
		return nil, f.lifecycleErr
	}
	return &dto.Problem{ProblemKey: problemKey, Status: status}, nil
}

func TestMapRepairSeverity(t *testing.T) {
	cases := []struct {
		name     string
		input    dto.RepairIssueSeverity
		expected dto.ProblemSeverity
	}{
		{"critical", dto.RepairIssueSeverities.REPAIRISSUESEVERITYCRITICAL, dto.ProblemSeverities.PROBLEMSEVERITYCRITICAL},
		{"error", dto.RepairIssueSeverities.REPAIRISSUESEVERITYERROR, dto.ProblemSeverities.PROBLEMSEVERITYERROR},
		{"warning", dto.RepairIssueSeverities.REPAIRISSUESEVERITYWARNING, dto.ProblemSeverities.PROBLEMSEVERITYWARNING},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, MapRepairSeverity(tc.input))
		})
	}
}

func TestMapRepairLifecycleStatus(t *testing.T) {
	cases := []struct {
		name     string
		input    dto.RepairLifecycleStatus
		expected dto.ProblemLifecycleStatus
	}{
		{"created", dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSCREATED, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED},
		{"ignored", dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSIGNORED, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSIGNORED},
		{"fixed", dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSFIXED, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED},
		{"dismissed", dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSDISMISSED, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDISMISSED},
		{"deleted", dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSDELETED, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDELETED},
		{"error", dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSERROR, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR},
		{"updated", dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSUPDATED, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSUPDATED},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, MapRepairLifecycleStatus(tc.input))
		})
	}
}

func TestBuildProblemFromCommand(t *testing.T) {
	cmd := dto.RepairCommandMessage{
		RepairID:       "r1",
		TranslationKey: "k1",
		Severity:       dto.RepairIssueSeverities.REPAIRISSUESEVERITYERROR,
		IsFixable:      true,
		IsPersistent:   true,
	}
	p := BuildProblemFromCommand(cmd, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED)
	require.NotNil(t, p)
	assert.Equal(t, "r1", p.ProblemKey)
	assert.Equal(t, "k1", p.Title)
	assert.Equal(t, dto.ProblemSeverities.PROBLEMSEVERITYERROR, p.Severity)

	cmd.TranslationKey = ""
	p = BuildProblemFromCommand(cmd, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSUPDATED)
	assert.Equal(t, "r1", p.Title)
}

func TestSyncBestEffort(t *testing.T) {
	ctx := context.Background()
	cmd := dto.RepairCommandMessage{RepairID: "r1", Severity: dto.RepairIssueSeverities.REPAIRISSUESEVERITYWARNING}
	store := &fakeStore{}
	SyncProblemFromCommand(ctx, store, cmd, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED)
	assert.Equal(t, 1, store.upserts)
	assert.Equal(t, "r1", store.lastProblem.ProblemKey)

	store.upsertErr = assert.AnError
	SyncProblemFromCommand(ctx, store, cmd, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSUPDATED)
	assert.Equal(t, 2, store.upserts)

	SyncProblemFromCommand(ctx, nil, cmd, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED)

	DismissProblem(ctx, store, "r1")
	assert.Equal(t, 1, store.dismisses)
	store.dismissErr = assert.AnError
	DismissProblem(ctx, store, "r1")
	assert.Equal(t, 2, store.dismisses)

	event := dto.RepairLifecycleMessage{RepairID: "r1", Status: dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSFIXED}
	ApplyProblemLifecycle(ctx, store, event)
	assert.Equal(t, 1, store.lifecycles)
	assert.Equal(t, dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED, store.lastStatus)
	store.lifecycleErr = assert.AnError
	ApplyProblemLifecycle(ctx, store, event)
	assert.Equal(t, 2, store.lifecycles)
}
