package mortgage

import "time"

type MortgageApplicationSubmitted struct {
	ApplicationID string    `json:"applicationId"`
	ApplicantName string    `json:"applicantName"`
	SubmittedAt   time.Time `json:"submittedAt"`
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
