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
)

type TimelineEntry struct {
	Step      string         `json:"step"`
	Status    TimelineStatus `json:"status"`
	Timestamp time.Time      `json:"timestamp"`
	Details   string         `json:"details,omitempty"`
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
}
