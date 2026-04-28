package mortgage

import (
	"time"

	"github.com/mrsimonemms/mortgage-application/mortgage-application/apps/worker/internal/mortgage/activities"
	"go.temporal.io/sdk/workflow"
)

// MortgageApplicationWorkflow orchestrates the full mortgage application happy path.
//
// Steps:
//  1. Intake - record receipt of the application
//  2. Credit check request - dispatch to external bureau (activity)
//  3. Durable wait - block until CreditCheckCompleted signal arrives (signal)
//  4. Offer reservation - allocate a mortgage offer
//  5. Complete application - mark the application as completed
func MortgageApplicationWorkflow(ctx workflow.Context, event MortgageApplicationSubmitted) (MortgageApplication, error) {
	app := MortgageApplication{
		ApplicationID: event.ApplicationID,
		ApplicantName: event.ApplicantName,
		Status:        StatusSubmitted,
		CurrentStep:   "submitted",
		CreatedAt:     workflow.Now(ctx),
		UpdatedAt:     workflow.Now(ctx),
		Timeline:      []TimelineEntry{},
	}

	if err := workflow.SetQueryHandler(ctx, QueryApplication, func() (MortgageApplication, error) {
		return app, nil
	}); err != nil {
		return app, err
	}

	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	})

	acts := activities.Activities{}

	recordTimeline(&app, ctx, "submitted", TimelineCompleted, "Application received", map[string]string{
		"applicationId": app.ApplicationID,
		"applicantName": app.ApplicantName,
	})

	if err := runIntake(ctx, actCtx, &app, acts); err != nil {
		return app, err
	}

	if err := requestCreditCheck(ctx, actCtx, &app, acts); err != nil {
		return app, err
	}

	creditResult := waitForCreditResult(ctx, &app)

	if creditResult.Result == CreditCheckRejected {
		app.Status = StatusRejected
		app.CurrentStep = "rejected"
		meta := map[string]string{"result": string(creditResult.Result)}
		if creditResult.Reference != "" {
			meta["reference"] = creditResult.Reference
		}
		recordTimeline(&app, ctx, "credit_check", TimelineCompleted, "Credit check rejected", meta)

		return app, nil
	}

	meta := map[string]string{"result": string(creditResult.Result)}
	if creditResult.Reference != "" {
		meta["reference"] = creditResult.Reference
	}
	recordTimeline(&app, ctx, "credit_check", TimelineCompleted, "Credit check approved", meta)

	if err := runOfferReservation(ctx, actCtx, &app, acts); err != nil {
		return app, err
	}

	if err := runCompleteApplication(ctx, actCtx, &app, acts); err != nil {
		return app, err
	}

	return app, nil
}

func runIntake(ctx, actCtx workflow.Context, app *MortgageApplication, acts activities.Activities) error {
	app.CurrentStep = "intake"
	recordTimeline(app, ctx, "intake", TimelineStarted, "Application intake started")

	var result activities.IntakeResult
	if err := workflow.ExecuteActivity(actCtx, acts.Intake, activities.IntakeInput{
		ApplicationID: app.ApplicationID,
		ApplicantName: app.ApplicantName,
	}).Get(ctx, &result); err != nil {
		return err
	}

	recordTimeline(app, ctx, "intake", TimelineCompleted, "Application intake recorded")

	return nil
}

func requestCreditCheck(ctx, actCtx workflow.Context, app *MortgageApplication, acts activities.Activities) error {
	app.Status = StatusCreditCheckPending
	app.CurrentStep = "credit_check"

	var result activities.CreditCheckRequestResult
	if err := workflow.ExecuteActivity(actCtx, acts.RequestCreditCheck, activities.CreditCheckInput{
		ApplicationID: app.ApplicationID,
	}).Get(ctx, &result); err != nil {
		return err
	}

	recordTimeline(app, ctx, "credit_check", TimelineStarted, "Credit and AML check requested", map[string]string{
		"reference": result.Reference,
	})

	return nil
}

// waitForCreditResult blocks the workflow durably until the CreditCheckCompleted signal
// arrives. A waiting entry is appended before blocking so the query handler reflects
// the durable pause while the workflow is suspended.
func waitForCreditResult(ctx workflow.Context, app *MortgageApplication) CreditCheckCompleted {
	app.CurrentStep = "awaiting_credit_result"
	recordTimeline(app, ctx, "credit_check", TimelineWaiting, "Awaiting credit bureau result")

	var result CreditCheckCompleted
	workflow.GetSignalChannel(ctx, CreditCheckCompletedSignal).Receive(ctx, &result)

	return result
}

func runOfferReservation(ctx, actCtx workflow.Context, app *MortgageApplication, acts activities.Activities) error {
	app.CurrentStep = "offer_reservation"
	recordTimeline(app, ctx, "offer_reservation", TimelineStarted, "Offer reservation started")

	var result activities.ReserveOfferResult
	if err := workflow.ExecuteActivity(actCtx, acts.ReserveOffer, activities.ReserveOfferInput{
		ApplicationID: app.ApplicationID,
	}).Get(ctx, &result); err != nil {
		return err
	}

	app.OfferID = result.OfferID
	app.Status = StatusOfferReserved
	recordTimeline(app, ctx, "offer_reservation", TimelineCompleted, "Offer reserved", map[string]string{
		"offerId": result.OfferID,
	})

	return nil
}

func runCompleteApplication(ctx, actCtx workflow.Context, app *MortgageApplication, acts activities.Activities) error {
	app.CurrentStep = "fulfilment"
	recordTimeline(app, ctx, "fulfilment", TimelineStarted, "Fulfilment started", map[string]string{
		"offerId": app.OfferID,
	})

	var result activities.CompleteApplicationResult
	if err := workflow.ExecuteActivity(actCtx, acts.CompleteApplication, activities.CompleteApplicationInput{
		ApplicationID: app.ApplicationID,
		OfferID:       app.OfferID,
	}).Get(ctx, &result); err != nil {
		return err
	}

	app.Status = StatusCompleted
	app.CurrentStep = "completed"
	recordTimeline(app, ctx, "fulfilment", TimelineCompleted, "Mortgage application completed", map[string]string{
		"offerId": app.OfferID,
		"status":  string(StatusCompleted),
	})

	return nil
}

// recordTimeline appends an audit entry to the application timeline and advances
// UpdatedAt. The optional metadata map carries structured data for the entry.
func recordTimeline(app *MortgageApplication, ctx workflow.Context, step string, status TimelineStatus, details string, metadata ...map[string]string) {
	entry := TimelineEntry{
		Step:      step,
		Status:    status,
		Timestamp: workflow.Now(ctx),
		Details:   details,
	}
	if len(metadata) > 0 {
		entry.Metadata = metadata[0]
	}
	app.Timeline = append(app.Timeline, entry)
	app.UpdatedAt = workflow.Now(ctx)
}
