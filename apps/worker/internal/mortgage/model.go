package mortgage

import "time"

type ApplicationStatus string

const (
	StatusSubmitted            ApplicationStatus = "submitted"
	StatusCreditCheckPending   ApplicationStatus = "credit_check_pending"
	StatusOfferReserved        ApplicationStatus = "offer_reserved"
	StatusCompleted            ApplicationStatus = "completed"
	StatusRejected             ApplicationStatus = "rejected"
	StatusCompensationRequired ApplicationStatus = "compensation_required"
	StatusCompensated          ApplicationStatus = "compensated"
)

type TimelineStatus string

const (
	TimelineStarted   TimelineStatus = "started"
	TimelineCompleted TimelineStatus = "completed"
	TimelineFailed    TimelineStatus = "failed"
	TimelineWaiting   TimelineStatus = "waiting"
)

// SLAStatus values are recorded against the mortgage application once the SLA
// outcome of an async wait has been finalised. They are durable so the UI and
// audit trail can show the SLA result after the transient deadline has been
// cleared.
const (
	SLAStatusWithin   = "within_sla"
	SLAStatusBreached = "sla_breached"
)

// TimelineEntry records a single transition in the mortgage application journey.
// Details holds a human-readable description. Metadata holds structured key/value
// data for the step (e.g. reference numbers, decision outcomes, IDs).
type TimelineEntry struct {
	Step      string            `json:"step"`
	Status    TimelineStatus    `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Details   string            `json:"details,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type MortgageApplication struct {
	ApplicationID string            `json:"applicationId"`
	ApplicantName string            `json:"applicantName"`
	Status        ApplicationStatus `json:"status"`
	CurrentStep   string            `json:"currentStep"`
	OfferID       string            `json:"offerId,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	Timeline      []TimelineEntry   `json:"timeline"`

	// SLA visibility for async dependencies. PendingDependency and PendingSince
	// are transient: they are only set while the workflow is durably waiting
	// on an external signal and are cleared once that wait resolves.
	PendingDependency *string    `json:"pendingDependency,omitempty"`
	PendingSince      *time.Time `json:"pendingSince,omitempty"`

	// SLADeadline, SLAStatus and SLABreached record the SLA outcome of the
	// most recent completed SLA-bounded wait. While the workflow is waiting
	// they describe the in-flight deadline and the query handler recomputes
	// SLABreached live; once the wait resolves they hold the durable outcome
	// so the UI and audit trail can show whether the SLA was met or breached.
	// They are reset together when a fresh wait begins (e.g. on operator
	// retry) so only the final attempt's outcome is retained.
	SLADeadline *time.Time `json:"slaDeadline,omitempty"`
	SLAStatus   *string    `json:"slaStatus,omitempty"`
	SLABreached *bool      `json:"slaBreached,omitempty"`
}
