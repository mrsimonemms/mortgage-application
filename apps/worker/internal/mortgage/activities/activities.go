package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

// Activities groups all mortgage application activity implementations.
// Register it as a single unit: w.RegisterActivity(&Activities{}).
type Activities struct{}

// Intake validates and records the receipt of a mortgage application.
func (Activities) Intake(_ context.Context, input IntakeInput) (IntakeResult, error) {
	if input.ApplicationID == "" {
		return IntakeResult{}, fmt.Errorf("intake failed: applicationId is required")
	}

	if input.ApplicantName == "" {
		return IntakeResult{}, fmt.Errorf("intake failed: applicantName is required")
	}

	return IntakeResult{
		ApplicationID: input.ApplicationID,
		ReceivedAt:    time.Now(),
	}, nil
}

// RequestCreditCheck submits a credit and AML check request to the external bureau.
// This activity only dispatches the request. The result is delivered asynchronously
// via the credit-check-completed signal sent through the API.
func (Activities) RequestCreditCheck(ctx context.Context, input CreditCheckInput) (CreditCheckRequestResult, error) {
	logger := activity.GetLogger(ctx)
	reference := "CREDIT-REQ-" + input.ApplicationID

	logger.Info("credit check requested; awaiting external result via signal",
		"applicationId", input.ApplicationID,
		"reference", reference,
	)

	return CreditCheckRequestResult{
		ApplicationID: input.ApplicationID,
		Reference:     reference,
	}, nil
}

// ReserveOffer allocates a mortgage offer for an approved application.
// The offer ID is derived deterministically from the application ID, making
// this activity idempotent: repeated calls for the same application always
// return the same offer. This also makes compensation straightforward: the
// offer ID is stable and can be passed directly to ReleaseOffer.
func (Activities) ReserveOffer(ctx context.Context, input ReserveOfferInput) (ReserveOfferResult, error) {
	logger := activity.GetLogger(ctx)
	offerID := "OFFER-" + input.ApplicationID

	logger.Info("offer reserved",
		"applicationId", input.ApplicationID,
		"offerId", offerID,
	)

	return ReserveOfferResult{
		ApplicationID: input.ApplicationID,
		OfferID:       offerID,
		ReservedAt:    time.Now(),
	}, nil
}

// ReleaseOffer cancels an existing offer reservation.
// This is the compensating action for ReserveOffer.
func (Activities) ReleaseOffer(ctx context.Context, input ReleaseOfferInput) (ReleaseOfferResult, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("offer released",
		"applicationId", input.ApplicationID,
		"offerId", input.OfferID,
	)

	return ReleaseOfferResult{
		ApplicationID: input.ApplicationID,
		ReleasedAt:    time.Now(),
	}, nil
}

// CompleteApplication finalises the mortgage once an offer has been reserved.
func (Activities) CompleteApplication(ctx context.Context, input FulfilmentInput) (FulfilmentResult, error) {
	logger := activity.GetLogger(ctx)

	logger.Info("mortgage application fulfilled",
		"applicationId", input.ApplicationID,
		"offerId", input.OfferID,
	)

	return FulfilmentResult{
		ApplicationID: input.ApplicationID,
		FulfilledAt:   time.Now(),
	}, nil
}
