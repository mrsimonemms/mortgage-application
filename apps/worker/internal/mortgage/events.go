package mortgage

import "time"

// WorkflowScenario controls which demo path the workflow executes.
type WorkflowScenario string

const (
	// ScenarioHappyPath runs the full successful mortgage workflow.
	ScenarioHappyPath WorkflowScenario = "happy_path"
	// ScenarioFailAfterOfferReservation reserves an offer then deliberately
	// fails at the completion stage, leaving the workflow in
	// StatusCompensationRequired for the compensation demo.
	ScenarioFailAfterOfferReservation WorkflowScenario = "fail_after_offer_reservation"
)

type MortgageApplicationSubmitted struct {
	ApplicationID string           `json:"applicationId"`
	ApplicantName string           `json:"applicantName"`
	SubmittedAt   time.Time        `json:"submittedAt"`
	Scenario      WorkflowScenario `json:"scenario,omitempty"`
}

type CreditCheckResult string

const (
	CreditCheckApproved CreditCheckResult = "approved"
	CreditCheckRejected CreditCheckResult = "rejected"
)

type CreditCheckCompleted struct {
	ApplicationID string            `json:"applicationId"`
	Result        CreditCheckResult `json:"result"`
	CompletedAt   time.Time         `json:"completedAt"`
	Reference     string            `json:"reference,omitempty"`
}

type OfferReserved struct {
	ApplicationID string    `json:"applicationId"`
	OfferID       string    `json:"offerId"`
	ReservedAt    time.Time `json:"reservedAt"`
}
