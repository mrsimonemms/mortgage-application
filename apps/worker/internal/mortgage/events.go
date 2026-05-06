package mortgage

import "time"

// CreditCheckSLA bounds how long the workflow can wait for the credit bureau
// signal before the SLA is considered breached. Kept small for demo visibility.
const CreditCheckSLA = 30 * time.Second

// PendingCreditCheck names the credit check dependency in query responses while
// the workflow is durably waiting for the credit bureau signal.
const PendingCreditCheck = "credit_check"

// WorkflowScenario controls which demo path the workflow executes.
type WorkflowScenario string

const (
	// ScenarioHappyPath runs the full successful mortgage workflow.
	ScenarioHappyPath WorkflowScenario = "happy_path"
	// ScenarioFailAfterOfferReservation reserves an offer then deliberately
	// fails at the completion stage. Temporal retries automatically; the
	// activity succeeds on the fifth attempt. Demonstrates retry-then-succeed.
	ScenarioFailAfterOfferReservation WorkflowScenario = "fail_after_offer_reservation"
	// ScenarioFailAndCompensate reserves an offer then fails at the completion
	// stage with a retry policy that is intentionally exhausted. The workflow
	// responds by releasing the reserved offer via compensation.
	ScenarioFailAndCompensate WorkflowScenario = "fail_and_compensate_after_offer_reservation"
)

type MortgageApplicationSubmitted struct {
	ApplicationID              string           `json:"applicationId"`
	ApplicantName              string           `json:"applicantName"`
	SubmittedAt                time.Time        `json:"submittedAt"`
	Scenario                   WorkflowScenario `json:"scenario,omitempty"`
	OriginalApplicationID      string           `json:"originalApplicationId,omitempty"`
	ExternalFailureRatePercent int              `json:"externalFailureRatePercent,omitempty"`
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
