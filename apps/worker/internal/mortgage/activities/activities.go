package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
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

// PerformPropertyValuation produces a deterministic stub valuation for the property
// associated with the application. In a production system this would call an external
// valuation service; here it returns a fixed amount for demo reliability.
func (Activities) PerformPropertyValuation(ctx context.Context, input PropertyValuationInput) (PropertyValuationResult, error) {
	logger := activity.GetLogger(ctx)
	reference := "VAL-" + input.ApplicationID

	logger.Info("property valuation completed",
		"applicationId", input.ApplicationID,
		"reference", reference,
	)

	return PropertyValuationResult{
		ApplicationID:      input.ApplicationID,
		ValuationReference: reference,
		ValuationAmount:    350000,
	}, nil
}

// CompleteApplication finalises the mortgage once an offer has been reserved.
//
// When SimulateFailure is set the activity fails on the first four attempts and
// succeeds on the fifth, demonstrating Temporal's automatic retry behaviour. Each
// failure is a retryable ApplicationError so Temporal drives the backoff — no manual
// retry loop is needed in workflow code.
func (Activities) CompleteApplication(ctx context.Context, input CompleteApplicationInput) (CompleteApplicationResult, error) {
	logger := activity.GetLogger(ctx)
	info := activity.GetInfo(ctx)

	if input.SimulateFailure && info.Attempt <= 4 {
		logger.Warn("simulating completion failure for demo; Temporal will retry",
			"applicationId", input.ApplicationID,
			"offerId", input.OfferID,
			"attempt", info.Attempt,
		)
		return CompleteApplicationResult{}, temporal.NewApplicationError(
			"completion failure injected for demo",
			"InjectedFulfilmentFailure",
			nil,
		)
	}

	logger.Info("mortgage application completed",
		"applicationId", input.ApplicationID,
		"offerId", input.OfferID,
		"attempt", info.Attempt,
	)

	return CompleteApplicationResult{
		ApplicationID: input.ApplicationID,
		CompletedAt:   time.Now(),
	}, nil
}
