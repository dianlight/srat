package dto

import "fmt"

// ProblemLifecycleMessage is emitted by clients/components to update Problem lifecycle state.
type ProblemLifecycleMessage struct {
	Type       string                 `json:"type"`
	ProblemKey string                 `json:"problem_key"`
	Status     ProblemLifecycleStatus `json:"status"`
	Error      *string                `json:"error,omitempty"`
}

// Validate checks ProblemLifecycleMessage required fields and value ranges.
func (msg ProblemLifecycleMessage) Validate() error {
	if msg.Type != ClientEventTypes.CLIENTEVENTTYPEPROBLEMLIFECYCLE.String() {
		return fmt.Errorf("invalid lifecycle type %q", msg.Type)
	}
	if msg.ProblemKey == "" {
		return fmt.Errorf("problem_key is required")
	}

	switch msg.Status {
	case ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSCREATED,
		ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSUPDATED,
		ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSIGNORED,
		ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSFIXED,
		ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDISMISSED,
		ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSDELETED,
		ProblemLifecycleStatuses.PROBLEMLIFECYCLESTATUSERROR:
		return nil
	default:
		return fmt.Errorf("invalid problem lifecycle status %q", msg.Status)
	}
}
