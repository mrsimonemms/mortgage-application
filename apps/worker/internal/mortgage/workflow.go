package mortgage

import (
	"strconv"
	"time"

	saga "github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/mortgage-application/mortgage-application/apps/worker/internal/mortgage/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowOptions configures the behaviour of the registered mortgage workflow.
// Set EnableValuation to false to emulate a pre-valuation (v1) worker.
type WorkflowOptions struct {
	EnableValuation bool
}

// NewMortgageApplicationWorkflow returns a workflow function bound to the given
// options. Register the returned closure using an explicit workflow name so the
// workflow type remains stable across worker versions:
//
//	w.RegisterWorkflowWithOptions(
//	    mortgage.NewMortgageApplicationWorkflow(opts),
//	    workflow.RegisterOptions{Name: "MortgageApplicationWorkflow"},
//	)
func NewMortgageApplicationWorkflow(opts WorkflowOptions) func(workflow.Context, MortgageApplicationSubmitted) (MortgageApplication, error) {
	return func(ctx workflow.Context, event MortgageApplicationSubmitted) (MortgageApplication, error) {
		return runMortgageApplicationWorkflow(ctx, event, opts)
	}
}

// runMortgageApplicationWorkflow orchestrates the full mortgage application.
//
// Steps:
//  1. Intake — record receipt of the application
//  2. Credit check request — dispatch to external bureau (activity)
//  3. Durable wait — block until CreditCheckCompleted signal arrives (signal)
//  4. Property valuation — assess the property value (opts.EnableValuation only)
//  5. Offer reservation — allocate a mortgage offer
//  6. Complete application — mark the application as completed
//
// Saga pattern: a compensation function is registered immediately after the offer
// reservation succeeds. If any later step fails and the workflow returns an error,
// the deferred compensator releases the reserved offer from a disconnected context
// and updates the audit trail. The workflow still returns the original error so the
// business failure is correctly reflected in Temporal.
//
// Versioning: when EnableValuation is true, step 4 is guarded by
// GetVersion("add-property-valuation"). Workflows started before this change skip
// the step on replay; new workflows execute it.
func runMortgageApplicationWorkflow(ctx workflow.Context, event MortgageApplicationSubmitted, opts WorkflowOptions) (app MortgageApplication, err error) {
	app = MortgageApplication{
		ApplicationID: event.ApplicationID,
		ApplicantName: event.ApplicantName,
		Status:        StatusSubmitted,
		CurrentStep:   "submitted",
		CreatedAt:     workflow.Now(ctx),
		UpdatedAt:     workflow.Now(ctx),
		Timeline:      []TimelineEntry{},
	}

	// The query handler returns a snapshot with an independent copy of the timeline
	// so callers cannot observe future mutations to the workflow's slice.
	if err = workflow.SetQueryHandler(ctx, QueryApplication, func() (MortgageApplication, error) {
		snapshot := app
		snapshot.Timeline = append([]TimelineEntry(nil), app.Timeline...)
		return snapshot, nil
	}); err != nil {
		return
	}

	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	})

	acts := activities.Activities{}

	// Compensation runs in LIFO order from a disconnected context whenever the
	// workflow is returning a non-nil error and at least one step registered a
	// compensation. The disconnected context ensures compensation activities are
	// not cancelled along with the failing workflow.
	var comp saga.Compensator
	defer func() {
		if err != nil {
			disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx)
			comp.Compensate(disconnectedCtx)
		}
	}()

	recordTimeline(&app, ctx, "submitted", TimelineCompleted, "Application received", map[string]string{
		"applicationId": app.ApplicationID,
		"applicantName": app.ApplicantName,
	})
	upsertSearchAttributes(ctx, &app, false)

	if err = runIntake(ctx, actCtx, &app, acts); err != nil {
		return
	}

	if err = requestCreditCheck(ctx, actCtx, &app, acts); err != nil {
		return
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
		upsertSearchAttributes(ctx, &app, false)
		return // err is nil; deferred comp is a no-op
	}

	meta := map[string]string{"result": string(creditResult.Result)}
	if creditResult.Reference != "" {
		meta["reference"] = creditResult.Reference
	}
	recordTimeline(&app, ctx, "credit_check", TimelineCompleted, "Credit check approved", meta)

	// EnableValuation gates the entire versioning block. When false (v1 worker) the
	// GetVersion call is skipped so no marker is written and old replay histories
	// remain valid. When true, GetVersion handles in-flight workflow safety: existing
	// executions without a marker return DefaultVersion and skip the step; new
	// executions return version 1 and execute it.
	if opts.EnableValuation {
		if v := workflow.GetVersion(ctx, "add-property-valuation", workflow.DefaultVersion, 1); v != workflow.DefaultVersion {
			if err = runPropertyValuation(ctx, actCtx, &app, acts); err != nil {
				return
			}
		}
	}

	if err = runOfferReservation(ctx, actCtx, &app, acts); err != nil {
		return
	}

	// Register compensation immediately after offer reservation succeeds.
	// Inputs are captured by value now so the closure does not read app fields
	// that may be mutated later (e.g. OfferID is cleared after compensation).
	registeredAppID := app.ApplicationID
	registeredOfferID := app.OfferID
	comp.Add(func(compCtx workflow.Context) error {
		return compensateReleaseOffer(compCtx, &app, acts, registeredAppID, registeredOfferID)
	})

	if err = runCompleteApplication(ctx, &app, acts, event.Scenario); err != nil {
		// Record the failure before the deferred compensator runs so the audit
		// trail shows the fulfilment failure ahead of the compensation entries.
		recordTimeline(&app, ctx, "fulfilment", TimelineFailed,
			"Fulfilment failed after maximum retries",
			map[string]string{
				"offerId": registeredOfferID,
				"reason":  "Maximum retry attempts exhausted",
			})
		app.Status = StatusCompensationRequired
		app.CurrentStep = "compensation"
		upsertSearchAttributes(ctx, &app, false)
		return // deferred compensator handles the release-offer step
	}

	return
}

func runIntake(ctx, actCtx workflow.Context, app *MortgageApplication, acts activities.Activities) error {
	app.CurrentStep = "intake"
	upsertSearchAttributes(ctx, app, false)
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
	upsertSearchAttributes(ctx, app, false)

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
// arrives. AwaitingExternalSignal is set true before blocking so the query handler
// and search attributes both reflect the durable pause while the workflow is suspended.
func waitForCreditResult(ctx workflow.Context, app *MortgageApplication) CreditCheckCompleted {
	app.CurrentStep = "awaiting_credit_result"
	recordTimeline(app, ctx, "credit_check", TimelineWaiting, "Awaiting credit bureau result")
	upsertSearchAttributes(ctx, app, true)

	var result CreditCheckCompleted
	workflow.GetSignalChannel(ctx, CreditCheckCompletedSignal).Receive(ctx, &result)

	upsertSearchAttributes(ctx, app, false)

	return result
}

func runPropertyValuation(ctx, actCtx workflow.Context, app *MortgageApplication, acts activities.Activities) error {
	app.CurrentStep = "property_valuation"
	recordTimeline(app, ctx, "valuation", TimelineStarted, "Property valuation started")
	upsertSearchAttributes(ctx, app, false)

	var result activities.PropertyValuationResult
	if err := workflow.ExecuteActivity(actCtx, acts.PerformPropertyValuation, activities.PropertyValuationInput{
		ApplicationID: app.ApplicationID,
	}).Get(ctx, &result); err != nil {
		return err
	}

	recordTimeline(app, ctx, "valuation", TimelineCompleted, "Property valuation completed", map[string]string{
		"valuationReference": result.ValuationReference,
		"valuationAmount":    strconv.FormatInt(result.ValuationAmount, 10),
	})
	upsertSearchAttributes(ctx, app, false)

	return nil
}

func runOfferReservation(ctx, actCtx workflow.Context, app *MortgageApplication, acts activities.Activities) error {
	app.CurrentStep = "offer_reservation"
	recordTimeline(app, ctx, "offer_reservation", TimelineStarted, "Offer reservation started")
	upsertSearchAttributes(ctx, app, false)

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
	upsertSearchAttributes(ctx, app, false)

	return nil
}

// runCompleteApplication executes the fulfilment step with scenario-specific retry
// behaviour. The retry policy and SimulateFailure flag are scoped to this step only
// so they do not affect any other activity in the workflow.
//
// fail_after_offer_reservation: fails on attempts 1–4, succeeds on attempt 5.
// fail_and_compensate_after_offer_reservation: fails on all 3 attempts, exhausting
// the retry policy. The caller is responsible for triggering compensation.
func runCompleteApplication(ctx workflow.Context, app *MortgageApplication, acts activities.Activities, scenario WorkflowScenario) error {
	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    10 * time.Second,
		MaximumAttempts:    5,
	}
	simulateFailure := false

	switch scenario {
	case ScenarioFailAfterOfferReservation:
		// Fails on attempts 1–4; succeeds on attempt 5.
		simulateFailure = true
	case ScenarioFailAndCompensate:
		// Fails on all 3 attempts, surfacing an error to the workflow.
		simulateFailure = true
		retryPolicy.MaximumAttempts = 3
	}

	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         retryPolicy,
	})

	app.CurrentStep = "fulfilment"
	recordTimeline(app, ctx, "fulfilment", TimelineStarted, "Fulfilment started", map[string]string{
		"offerId": app.OfferID,
	})
	upsertSearchAttributes(ctx, app, false)

	var result activities.CompleteApplicationResult
	if err := workflow.ExecuteActivity(actCtx, acts.CompleteApplication, activities.CompleteApplicationInput{
		ApplicationID:   app.ApplicationID,
		OfferID:         app.OfferID,
		SimulateFailure: simulateFailure,
	}).Get(ctx, &result); err != nil {
		return err
	}

	app.Status = StatusCompleted
	app.CurrentStep = "completed"
	recordTimeline(app, ctx, "fulfilment", TimelineCompleted, "Mortgage application completed", map[string]string{
		"offerId": app.OfferID,
		"status":  string(StatusCompleted),
	})
	upsertSearchAttributes(ctx, app, false)

	return nil
}

// compensateReleaseOffer is the compensation action for a successful offer reservation.
// It runs from a disconnected context so it is not cancelled when the parent workflow
// context is cancelled on failure. applicationID and offerID are passed explicitly
// rather than read from app to ensure the correct values are used even if app is
// mutated between registration and execution.
func compensateReleaseOffer(ctx workflow.Context, app *MortgageApplication, acts activities.Activities, applicationID, offerID string) error {
	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	})

	recordTimeline(app, ctx, "compensation", TimelineStarted,
		"Initiating compensation: releasing reserved offer",
		map[string]string{"offerId": offerID})

	var result activities.ReleaseOfferResult
	if err := workflow.ExecuteActivity(actCtx, acts.ReleaseOffer, activities.ReleaseOfferInput{
		ApplicationID: applicationID,
		OfferID:       offerID,
	}).Get(ctx, &result); err != nil {
		return err
	}

	// Clear the offer ID: the offer is no longer reserved.
	app.OfferID = ""
	app.Status = StatusCompensated
	app.CurrentStep = "compensated"
	recordTimeline(app, ctx, "compensation", TimelineCompleted,
		"Compensation complete: offer released",
		map[string]string{"offerId": offerID, "status": string(StatusCompensated)})
	upsertSearchAttributes(ctx, app, false)

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

// upsertSearchAttributes syncs the four custom search attributes to current workflow
// state. awaitingSignal must be passed explicitly as it is not stored on
// MortgageApplication. Failures are logged as warnings; they do not abort the workflow.
func upsertSearchAttributes(ctx workflow.Context, app *MortgageApplication, awaitingSignal bool) {
	if err := workflow.UpsertTypedSearchAttributes(
		ctx,
		saApplicationStatus.ValueSet(string(app.Status)),
		saCurrentStep.ValueSet(app.CurrentStep),
		saHasOffer.ValueSet(app.OfferID != ""),
		saAwaitingExternalSignal.ValueSet(awaitingSignal),
	); err != nil {
		workflow.GetLogger(ctx).Warn("failed to upsert search attributes", "error", err)
	}
}
