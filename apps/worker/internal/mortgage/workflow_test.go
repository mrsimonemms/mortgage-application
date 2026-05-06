package mortgage

import (
	"testing"
	"time"

	"github.com/mrsimonemms/mortgage-application/apps/worker/internal/mortgage/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

// runHappyPath executes the full mortgage workflow through the Temporal test environment
// and returns the final application state. The credit check signal is delivered with a
// short delay so the workflow can dispatch its upstream activities first.
func runHappyPath(t *testing.T) MortgageApplication {
	t.Helper()

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	if !assert.True(t, env.IsWorkflowCompleted(), "workflow should have completed") {
		return MortgageApplication{}
	}
	if !assert.NoError(t, env.GetWorkflowError()) {
		return MortgageApplication{}
	}

	var result MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&result))
	return result
}

// TestMortgageApplicationWorkflow_HappyPath confirms the final application state and
// the ordered sequence of timeline steps after a successful run.
func TestMortgageApplicationWorkflow_HappyPath(t *testing.T) {
	result := runHappyPath(t)

	assert.Equal(t, StatusCompleted, result.Status)
	assert.Equal(t, "completed", result.CurrentStep)
	assert.Equal(t, testApplicationID, result.ApplicationID)
	assert.Equal(t, testApplicantName, result.ApplicantName)
	assert.NotEmpty(t, result.OfferID)

	steps := make([]string, len(result.Timeline))
	for i, e := range result.Timeline {
		steps[i] = e.Step + "/" + string(e.Status)
	}

	assert.Equal(t, []string{
		"submitted/completed",
		"intake/started",
		"intake/completed",
		"credit_check/started",
		"credit_check/waiting",
		"credit_check/completed",
		"offer_reservation/started",
		"offer_reservation/completed",
		"fulfilment/started",
		"fulfilment/completed",
		"notification/started",
		"notification/completed",
	}, steps)
}

// TestMortgageApplicationWorkflow_AuditTrail verifies that each timeline entry carries
// a non-zero timestamp, a human-readable description, and the expected structured
// metadata for steps that produce meaningful outputs.
func TestMortgageApplicationWorkflow_AuditTrail(t *testing.T) {
	result := runHappyPath(t)

	// All entries must have a non-zero timestamp.
	for i, e := range result.Timeline {
		assert.False(t, e.Timestamp.IsZero(), "entry %d (%s/%s) has zero timestamp", i, e.Step, e.Status)
	}

	// Build a lookup by step+status for targeted content assertions.
	byKey := make(map[string]TimelineEntry, len(result.Timeline))
	for _, e := range result.Timeline {
		byKey[e.Step+"/"+string(e.Status)] = e
	}

	cases := []struct {
		key      string
		details  string
		metadata map[string]string
	}{
		{
			key:     "submitted/completed",
			details: "Application received",
			metadata: map[string]string{
				"applicationId": testApplicationID,
				"applicantName": testApplicantName,
			},
		},
		{
			key:     "intake/started",
			details: "Application intake started",
		},
		{
			key:     "intake/completed",
			details: "Application intake recorded",
		},
		{
			key:     "credit_check/started",
			details: "Credit and AML check requested",
			metadata: map[string]string{
				"reference": "CREDIT-REQ-" + testApplicationID,
			},
		},
		{
			key:     "credit_check/waiting",
			details: "Awaiting credit bureau result",
		},
		{
			key:     "credit_check/completed",
			details: "Credit check approved",
			metadata: map[string]string{
				"result": "approved",
			},
		},
		{
			key:     "offer_reservation/started",
			details: "Offer reservation started",
		},
		{
			key:     "offer_reservation/completed",
			details: "Offer reserved",
			metadata: map[string]string{
				"offerId": "OFFER-" + testApplicationID,
			},
		},
		{
			key:     "fulfilment/started",
			details: "Fulfilment started",
			metadata: map[string]string{
				"offerId": "OFFER-" + testApplicationID,
			},
		},
		{
			key:     "fulfilment/completed",
			details: "Mortgage application completed",
			metadata: map[string]string{
				"offerId": "OFFER-" + testApplicationID,
				"status":  "completed",
			},
		},
		{
			key:     "notification/started",
			details: "Notifying applicant of final outcome",
			metadata: map[string]string{
				"applicationId": testApplicationID,
				"status":        string(NotificationApproved),
			},
		},
		{
			key:     "notification/completed",
			details: "Notification dispatched to applicant",
			metadata: map[string]string{
				"applicationId": testApplicationID,
				"status":        string(NotificationApproved),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			entry, ok := byKey[tc.key]
			assert.True(t, ok, "timeline entry %q not found", tc.key)
			assert.Equal(t, tc.details, entry.Details)
			for k, v := range tc.metadata {
				assert.Equal(t, v, entry.Metadata[k], "metadata[%q] for entry %q", k, tc.key)
			}
		})
	}
}

// TestMortgageApplicationWorkflow_QueryWhileWaiting queries the workflow mid-flight
// while it is blocked on the credit bureau signal. The response must show the
// awaiting_credit_result step and include a credit_check/waiting timeline entry.
// While blocked, SLA visibility fields must be populated and SLAStatus must be
// nil (outcome not yet finalised). After completion, PendingDependency and
// PendingSince must be cleared, but the durable SLA outcome (SLADeadline,
// SLAStatus, SLABreached) must remain so the UI can show whether the SLA was
// met or breached.
func TestMortgageApplicationWorkflow_QueryWhileWaiting(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	env.RegisterDelayedCallback(func() {
		val, err := env.QueryWorkflow(QueryApplication)
		assert.NoError(t, err)

		var app MortgageApplication
		assert.NoError(t, val.Get(&app))

		assert.Equal(t, "awaiting_credit_result", app.CurrentStep)
		assert.Equal(t, StatusCreditCheckPending, app.Status)

		var found bool
		for _, e := range app.Timeline {
			if e.Step == "credit_check" && e.Status == TimelineWaiting {
				found = true
			}
		}
		assert.True(t, found, "timeline should include credit_check/waiting entry while blocked")

		if assert.NotNil(t, app.PendingDependency, "pendingDependency must be set while waiting") {
			assert.Equal(t, PendingCreditCheck, *app.PendingDependency)
		}
		assert.NotNil(t, app.PendingSince, "pendingSince must be set while waiting")
		if assert.NotNil(t, app.SLADeadline, "slaDeadline must be set while waiting") {
			assert.Equal(t, app.PendingSince.Add(CreditCheckSLA), *app.SLADeadline)
		}
		assert.NotNil(t, app.SLABreached, "slaBreached must be reported while waiting")
		assert.Nil(t, app.SLAStatus, "slaStatus must be nil while the SLA outcome is in flight")

		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var final MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&final))
	assert.Nil(t, final.PendingDependency, "pendingDependency must be cleared after wait resolves")
	assert.Nil(t, final.PendingSince, "pendingSince must be cleared after wait resolves")
	assert.NotNil(t, final.SLADeadline, "slaDeadline must persist as part of the durable SLA outcome")
	if assert.NotNil(t, final.SLAStatus, "slaStatus must record the durable outcome") {
		assert.Equal(t, SLAStatusWithin, *final.SLAStatus)
	}
	if assert.NotNil(t, final.SLABreached, "slaBreached must record the durable outcome") {
		assert.False(t, *final.SLABreached, "credit check signalled within SLA window")
	}
}

// TestMortgageApplicationWorkflow_RejectedCreditCheck confirms the final state and
// timeline when the credit bureau returns a rejected result.
func TestMortgageApplicationWorkflow_RejectedCreditCheck(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckRejected,
			Reference:     "REF-REJECTED",
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, StatusRejected, result.Status)
	assert.Equal(t, "rejected", result.CurrentStep)
	assert.Empty(t, result.OfferID)

	byKey := make(map[string]TimelineEntry, len(result.Timeline))
	for _, e := range result.Timeline {
		byKey[e.Step+"/"+string(e.Status)] = e
	}

	rejection, ok := byKey["credit_check/completed"]
	assert.True(t, ok, "expected credit_check/completed entry in timeline")
	assert.Equal(t, "Credit check rejected", rejection.Details)
	assert.Equal(t, "rejected", rejection.Metadata["result"])
	assert.Equal(t, "REF-REJECTED", rejection.Metadata["reference"])

	// Rejection is a normal terminal outcome and must produce a notification
	// addressed to the application with status "rejected".
	notification, ok := byKey["notification/completed"]
	if assert.True(t, ok, "rejected applications must still produce a notification") {
		assert.Equal(t, testApplicationID, notification.Metadata["applicationId"])
		assert.Equal(t, string(NotificationRejected), notification.Metadata["status"])
	}
}

// TestMortgageApplicationWorkflow_RetryAndSucceed verifies the fail_after_offer_reservation
// scenario. CompleteApplication is invoked with SimulateFailure set, which makes the
// activity fail on attempts 1–4 and succeed on attempt 5. Temporal drives the retries
// automatically with exponential backoff. No error is swallowed in workflow code: if all
// retries were exhausted the error would propagate normally. The workflow must complete
// successfully with StatusCompleted after the fifth attempt succeeds.
func TestMortgageApplicationWorkflow_RetryAndSucceed(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Scenario:      ScenarioFailAfterOfferReservation,
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError(), "workflow must complete without error after retries succeed")

	var result MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, StatusCompleted, result.Status)
	assert.Equal(t, "completed", result.CurrentStep)
	assert.NotEmpty(t, result.OfferID, "offerId must remain present")

	byKey := make(map[string]TimelineEntry, len(result.Timeline))
	for _, e := range result.Timeline {
		byKey[e.Step+"/"+string(e.Status)] = e
	}

	_, hasStarted := byKey["fulfilment/started"]
	assert.True(t, hasStarted, "audit trail must include fulfilment/started")

	_, hasCompleted := byKey["fulfilment/completed"]
	assert.True(t, hasCompleted, "audit trail must include fulfilment/completed after successful retry")
}

// TestMortgageApplicationWorkflow_Compensation verifies the
// fail_and_compensate_after_offer_reservation scenario following the saga pattern.
//
// Saga behaviour under test:
//   - Offer reservation succeeds; compensation is registered immediately.
//   - CompleteApplication fails on all 3 retry attempts (retries exhausted).
//   - The workflow records the failure, then the deferred compensator runs
//     ReleaseOffer from a disconnected context.
//   - The workflow returns a non-nil error (the business transaction failed).
//   - The final application state — accessible via the query handler — reflects
//     the compensated terminal state: StatusCompensated, OfferID cleared.
func TestMortgageApplicationWorkflow_Compensation(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Scenario:      ScenarioFailAndCompensate,
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	// The workflow must return an error: compensation does not convert the failure
	// into a success. The business transaction failed; the saga cleaned up.
	assert.Error(t, env.GetWorkflowError(), "workflow must return an error after compensation")

	// The query handler is still accessible after workflow completion. It returns
	// the state as it was when the workflow last mutated app — i.e. after the
	// deferred compensator updated Status, CurrentStep and OfferID.
	val, queryErr := env.QueryWorkflow(QueryApplication)
	assert.NoError(t, queryErr, "query must succeed even after workflow failure")

	var result MortgageApplication
	assert.NoError(t, val.Get(&result))

	assert.Equal(t, StatusCompensated, result.Status)
	assert.Equal(t, "compensated", result.CurrentStep)
	assert.Empty(t, result.OfferID, "offerId must be cleared once the offer is released")

	byKey := make(map[string]TimelineEntry, len(result.Timeline))
	for _, e := range result.Timeline {
		byKey[e.Step+"/"+string(e.Status)] = e
	}

	// Offer reservation must succeed before the failure.
	offerReserved, ok := byKey["offer_reservation/completed"]
	assert.True(t, ok, "audit trail must include offer_reservation/completed")
	assert.Equal(t, "OFFER-"+testApplicationID, offerReserved.Metadata["offerId"])

	// Fulfilment must be attempted and recorded as started before failing.
	fulfilmentStarted, ok := byKey["fulfilment/started"]
	assert.True(t, ok, "audit trail must include fulfilment/started")
	assert.Equal(t, "OFFER-"+testApplicationID, fulfilmentStarted.Metadata["offerId"])

	// Fulfilment failure must appear in the timeline before compensation entries.
	fulfilmentFailed, ok := byKey["fulfilment/failed"]
	assert.True(t, ok, "audit trail must include fulfilment/failed after retry exhaustion")
	assert.Equal(t, "OFFER-"+testApplicationID, fulfilmentFailed.Metadata["offerId"])
	assert.Equal(t, "Maximum retry attempts exhausted", fulfilmentFailed.Metadata["reason"])

	// Compensation must be recorded as started with the offer ID.
	compStarted, ok := byKey["compensation/started"]
	assert.True(t, ok, "audit trail must include compensation/started")
	assert.Equal(t, "OFFER-"+testApplicationID, compStarted.Metadata["offerId"])

	// Compensation must complete and record the terminal state.
	compCompleted, ok := byKey["compensation/completed"]
	assert.True(t, ok, "audit trail must include compensation/completed")
	assert.Equal(t, "OFFER-"+testApplicationID, compCompleted.Metadata["offerId"])
	assert.Equal(t, string(StatusCompensated), compCompleted.Metadata["status"])

	// A compensated saga outcome must NOT trigger the final applicant
	// notification. The applicant should not be told their mortgage was
	// approved when the saga has rolled the offer back.
	for _, e := range result.Timeline {
		assert.NotEqual(t, "notification", e.Step,
			"compensated workflows must not produce any notification audit entry")
	}
}

// TestMortgageApplicationWorkflow_RetryCreditCheck verifies that when an operator
// sends the RetryCreditCheckSignal the workflow records the operator_retry_credit_check
// audit event, re-requests the credit check, and completes normally when the
// CreditCheckCompleted signal subsequently arrives.
func TestMortgageApplicationWorkflow_RetryCreditCheck(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// First, send a retry signal while the workflow is waiting for credit result.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(RetryCreditCheckSignal, nil)
	}, time.Second)

	// Then deliver the actual credit result after the retry loop restarts.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC),
		})
	}, 2*time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, StatusCompleted, result.Status)

	var foundRetry bool
	for _, e := range result.Timeline {
		if e.Step == "operator_retry_credit_check" {
			foundRetry = true
			assert.Equal(t, testApplicationID, e.Metadata["applicationId"])
		}
	}
	assert.True(t, foundRetry, "timeline must include operator_retry_credit_check entry")
}

// TestMortgageApplicationWorkflow_Rerun verifies that a workflow started with
// OriginalApplicationID set records the operator_rerun_application audit entry and
// otherwise completes the standard happy path.
func TestMortgageApplicationWorkflow_Rerun(t *testing.T) {
	const newAppID = "new-app-id"
	const originalAppID = "original-app-id"

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID:         newAppID,
		ApplicantName:         testApplicantName,
		SubmittedAt:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		OriginalApplicationID: originalAppID,
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: newAppID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, StatusCompleted, result.Status)

	var foundRerun bool
	for _, e := range result.Timeline {
		if e.Step == "operator_rerun_application" {
			foundRerun = true
			assert.Equal(t, originalAppID, e.Metadata["originalApplicationId"])
		}
	}
	assert.True(t, foundRerun, "timeline must include operator_rerun_application entry")
}

// TestSearchAttributeKeys_Names verifies that the search attribute key names match
// the strings that must be registered with the Temporal server.
func TestSearchAttributeKeys_Names(t *testing.T) {
	cases := []struct {
		name string
		got  string
	}{
		{"ApplicationStatus", saApplicationStatus.GetName()},
		{"CurrentStep", saCurrentStep.GetName()},
		{"HasOffer", saHasOffer.GetName()},
		{"AwaitingExternalSignal", saAwaitingExternalSignal.GetName()},
		{"WithinSLA", saWithinSLA.GetName()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.name, tc.got)
		})
	}
}

// TestMortgageApplicationWorkflow_SLABreached drives the wait long enough that
// the credit check signal arrives after the SLA deadline. The final application
// must report SLAStatus="sla_breached" and SLABreached=true durably, even
// though the workflow continued through the rest of the happy path.
func TestMortgageApplicationWorkflow_SLABreached(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Deliver the credit signal after the SLA deadline elapses. CreditCheckSLA
	// is 30s, so a 60s delay guarantees the deadline has passed.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, 60*time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var final MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&final))

	assert.Equal(t, StatusCompleted, final.Status, "workflow still completes after SLA breach")
	if assert.NotNil(t, final.SLAStatus, "slaStatus must be persisted") {
		assert.Equal(t, SLAStatusBreached, *final.SLAStatus)
	}
	if assert.NotNil(t, final.SLABreached, "slaBreached must be persisted") {
		assert.True(t, *final.SLABreached, "signal arrived after the deadline")
	}
	assert.NotNil(t, final.SLADeadline, "slaDeadline is retained as part of the durable outcome")
	assert.Nil(t, final.PendingDependency, "pendingDependency must be cleared after wait resolves")
	assert.Nil(t, final.PendingSince, "pendingSince must be cleared after wait resolves")
}

// TestMortgageApplicationWorkflow_SLARetryResetsTracking confirms that an
// operator retry replaces the in-flight SLA tracking and that only the final
// successful attempt's outcome is persisted.
func TestMortgageApplicationWorkflow_SLARetryResetsTracking(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Trigger a retry well after the first attempt's SLA window has elapsed,
	// then deliver the credit result quickly inside the new attempt's window.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(RetryCreditCheckSignal, nil)
	}, 60*time.Second)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 2, 0, 0, time.UTC),
		})
	}, 61*time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var final MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&final))

	if assert.NotNil(t, final.SLAStatus, "final attempt must record an SLA outcome") {
		assert.Equal(t, SLAStatusWithin, *final.SLAStatus,
			"only the final attempt's outcome is retained; that attempt resolved within its own SLA")
	}
	if assert.NotNil(t, final.SLABreached) {
		assert.False(t, *final.SLABreached)
	}
}

// TestMortgageApplicationWorkflow_NotificationApprovedPayload verifies that
// the SendNotification activity is invoked with the applicationId and the
// "approved" status when the workflow reaches the normal successful terminal
// outcome.
func TestMortgageApplicationWorkflow_NotificationApprovedPayload(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	var captured activities.SendNotificationInput
	env.OnActivity(activities.Activities{}.SendNotification, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(activities.SendNotificationInput)
		}).
		Return(activities.SendNotificationResult{
			ApplicationID: testApplicationID,
			Status:        string(NotificationApproved),
			DeliveredAt:   time.Date(2024, 1, 1, 0, 5, 0, 0, time.UTC),
		}, nil)

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	assert.Equal(t, testApplicationID, captured.ApplicationID,
		"notification must address the applicationId")
	assert.Equal(t, string(NotificationApproved), captured.Status)
}

// TestMortgageApplicationWorkflow_NotificationFailureDoesNotCompensate
// confirms the resilience contract for the final notification step: even when
// the activity fails with a non-retryable error, the workflow must still
// complete successfully (StatusCompleted, OfferID retained) and must NOT
// trigger the saga compensator. A failed notification is a soft failure
// recorded in the audit trail; it does not roll back a fulfilled mortgage.
func TestMortgageApplicationWorkflow_NotificationFailureDoesNotCompensate(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	env.OnActivity(activities.Activities{}.SendNotification, mock.Anything, mock.Anything).
		Return(activities.SendNotificationResult{}, temporal.NewNonRetryableApplicationError(
			"simulated permanent notification failure",
			"NotificationFailure",
			nil,
		))

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError(),
		"a notification failure must not cause the workflow itself to fail")

	var result MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, StatusCompleted, result.Status,
		"workflow must remain in completed state; notification failure is non-fatal")
	assert.NotEmpty(t, result.OfferID,
		"offer must remain reserved; notification failure must not trigger compensation")

	// The audit trail must record the notification failure so operators can
	// see what happened, while no compensation entries should appear.
	var sawNotificationFailed, sawCompensation bool
	for _, e := range result.Timeline {
		if e.Step == "notification" && e.Status == TimelineFailed {
			sawNotificationFailed = true
		}
		if e.Step == "compensation" {
			sawCompensation = true
		}
	}
	assert.True(t, sawNotificationFailed, "audit trail must include notification/failed")
	assert.False(t, sawCompensation, "compensation must NOT run for a notification failure")
}
