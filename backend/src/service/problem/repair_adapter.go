// Package problem centralizes Repair to Problem mirroring behind a single facade.
package problem

import (
	"context"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/tlog"
)

// Store is the minimal ProblemService surface needed for repair mirroring.
// Defined locally so service can import this package without an import cycle.
type Store interface {
	Upsert(problem *dto.Problem) (*dto.Problem, error)
	Dismiss(problemKey string) error
	ApplyLifecycle(problemKey string, status dto.ProblemLifecycleStatus, lastError *string) (*dto.Problem, error)
}

// MapRepairSeverity maps a repair severity to the unified problem severity.
func MapRepairSeverity(severity dto.RepairIssueSeverity) dto.ProblemSeverity {
	switch severity {
	case dto.RepairIssueSeverities.REPAIRISSUESEVERITYCRITICAL:
		return dto.ProblemSeverities.PROBLEMSEVERITYCRITICAL
	case dto.RepairIssueSeverities.REPAIRISSUESEVERITYERROR:
		return dto.ProblemSeverities.PROBLEMSEVERITYERROR
	default:
		return dto.ProblemSeverities.PROBLEMSEVERITYWARNING
	}
}

// MapRepairLifecycleStatus maps a repair lifecycle status to the problem status.
func MapRepairLifecycleStatus(status dto.RepairLifecycleStatus) dto.ProblemLifecycleStatus {
	switch status {
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSCREATED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSIGNORED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSIGNORED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSFIXED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSDISMISSED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDISMISSED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSDELETED:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDELETED
	case dto.RepairLifecycleStatuses.REPAIRLIFECYCLESTATUSERROR:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR
	default:
		return dto.ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSUPDATED
	}
}

// BuildProblemFromCommand builds the mirrored problem payload for a repair command.
func BuildProblemFromCommand(command dto.RepairCommandMessage, status dto.ProblemLifecycleStatus) *dto.Problem {
	title := command.TranslationKey
	if title == "" {
		title = command.RepairID
	}
	return &dto.Problem{
		ProblemKey:              command.RepairID,
		Title:                   title,
		Description:             title,
		Severity:                MapRepairSeverity(command.Severity),
		Status:                  status,
		TranslationKey:          command.TranslationKey,
		TranslationPlaceholders: command.TranslationPlaceholders,
		Data:                    command.Data,
		LearnMoreURL:            command.LearnMoreURL,
		IsFixable:               command.IsFixable,
		IsPersistent:            command.IsPersistent,
	}
}

// SyncProblemFromCommand mirrors a repair command into the problem store.
// Best-effort warn-only semantics: mirroring failures never fail the caller.
func SyncProblemFromCommand(ctx context.Context, store Store, command dto.RepairCommandMessage, status dto.ProblemLifecycleStatus) {
	if store == nil {
		return
	}
	if _, err := store.Upsert(BuildProblemFromCommand(command, status)); err != nil {
		tlog.WarnContext(ctx, "Failed to sync repair command to problem", "repair_id", command.RepairID, "error", err)
	}
}

// DismissProblem removes the mirrored problem for a deleted repair. Best-effort.
func DismissProblem(ctx context.Context, store Store, repairID string) {
	if store == nil {
		return
	}
	if err := store.Dismiss(repairID); err != nil {
		tlog.WarnContext(ctx, "Failed to dismiss mirrored problem", "problem_key", repairID, "error", err)
	}
}

// ApplyProblemLifecycle mirrors a repair lifecycle event. Best-effort.
func ApplyProblemLifecycle(ctx context.Context, store Store, event dto.RepairLifecycleMessage) {
	if store == nil {
		return
	}
	if _, err := store.ApplyLifecycle(event.RepairID, MapRepairLifecycleStatus(event.Status), event.Error); err != nil {
		tlog.WarnContext(ctx, "Failed to sync repair lifecycle to problem", "repair_id", event.RepairID, "status", event.Status, "error", err)
	}
}
