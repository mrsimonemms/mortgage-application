package activities

import "time"

type IntakeInput struct {
	ApplicationID string `json:"applicationId"`
	ApplicantName string `json:"applicantName"`
}

type IntakeResult struct {
	ApplicationID string    `json:"applicationId"`
	ReceivedAt    time.Time `json:"receivedAt"`
}

type CreditCheckInput struct {
	ApplicationID              string `json:"applicationId"`
	ExternalFailureRatePercent int    `json:"externalFailureRatePercent,omitempty"`
}

// CreditCheckRequestResult is returned by RequestCreditCheck to confirm the
// request was dispatched and to provide a correlation reference.
type CreditCheckRequestResult struct {
	ApplicationID string `json:"applicationId"`
	Reference     string `json:"reference"`
}

type CreditCheckOutput struct {
	ApplicationID string    `json:"applicationId"`
	Result        string    `json:"result"`
	Reference     string    `json:"reference,omitempty"`
	CompletedAt   time.Time `json:"completedAt"`
}

type ReserveOfferInput struct {
	ApplicationID              string `json:"applicationId"`
	ExternalFailureRatePercent int    `json:"externalFailureRatePercent,omitempty"`
}

type ReserveOfferResult struct {
	ApplicationID string    `json:"applicationId"`
	OfferID       string    `json:"offerId"`
	ReservedAt    time.Time `json:"reservedAt"`
}

type CompleteApplicationInput struct {
	ApplicationID string `json:"applicationId"`
	OfferID       string `json:"offerId"`
	// SimulateFailure causes the activity to fail on the first four attempts and
	// succeed on the fifth, demonstrating Temporal's automatic retry behaviour.
	// Used for the fail_after_offer_reservation demo scenario only.
	SimulateFailure            bool `json:"simulateFailure,omitempty"`
	ExternalFailureRatePercent int  `json:"externalFailureRatePercent,omitempty"`
}

type CompleteApplicationResult struct {
	ApplicationID string    `json:"applicationId"`
	CompletedAt   time.Time `json:"completedAt"`
}

type ReleaseOfferInput struct {
	ApplicationID              string `json:"applicationId"`
	OfferID                    string `json:"offerId"`
	ExternalFailureRatePercent int    `json:"externalFailureRatePercent,omitempty"`
}

type ReleaseOfferResult struct {
	ApplicationID string    `json:"applicationId"`
	ReleasedAt    time.Time `json:"releasedAt"`
}

// SendNotificationInput carries the data required to dispatch the final
// applicant notification. ApplicationID identifies the recipient; Status is
// the terminal outcome of the workflow ("approved" or "rejected").
// Compensated outcomes do not produce a notification so are not represented
// here.
type SendNotificationInput struct {
	ApplicationID              string `json:"applicationId"`
	Status                     string `json:"status"`
	ExternalFailureRatePercent int    `json:"externalFailureRatePercent,omitempty"`
}

type SendNotificationResult struct {
	ApplicationID string    `json:"applicationId"`
	Status        string    `json:"status"`
	DeliveredAt   time.Time `json:"deliveredAt"`
}
