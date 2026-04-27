package activities

import (
	"context"
	"fmt"
	"time"
)

// Activities groups all mortgage application activity implementations.
// Register it as a single unit: w.RegisterActivity(&Activities{}).
type Activities struct{}

// Intake records the receipt of a mortgage application.
func (Activities) Intake(_ context.Context, input IntakeInput) (IntakeResult, error) {
	return IntakeResult{
		ApplicationID: input.ApplicationID,
		ReceivedAt:    time.Now(),
	}, nil
}

// RequestCreditCheck submits a credit and AML check request to the external bureau.
// This activity only dispatches the request. The result is delivered later via a
// CreditCheckCompleted signal sent through the API.
func (Activities) RequestCreditCheck(_ context.Context, input CreditCheckInput) error {
	// In a real system this would dispatch to an external credit bureau.
	// For the demo, the operator sends the result back via the API signal endpoint.
	fmt.Printf("credit check requested for application %s\n", input.ApplicationID)

	return nil
}

// ReserveOffer allocates a mortgage offer for an approved application.
func (Activities) ReserveOffer(_ context.Context, input ReserveOfferInput) (ReserveOfferResult, error) {
	offerID := "OFFER-" + input.ApplicationID

	return ReserveOfferResult{
		ApplicationID: input.ApplicationID,
		OfferID:       offerID,
		ReservedAt:    time.Now(),
	}, nil
}

// CompleteApplication finalises the mortgage once an offer has been reserved.
func (Activities) CompleteApplication(_ context.Context, input FulfilmentInput) (FulfilmentResult, error) {
	return FulfilmentResult{
		ApplicationID: input.ApplicationID,
		FulfilledAt:   time.Now(),
	}, nil
}
