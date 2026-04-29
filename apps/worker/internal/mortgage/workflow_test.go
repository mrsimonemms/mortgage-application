package mortgage

import (
	"testing"
	"time"

	"github.com/mrsimonemms/mortgage-application/mortgage-application/apps/worker/internal/mortgage/activities"
	"github.com/stretchr/testify/assert"
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

		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
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
// fail_and_compensate_after_offer_reservation scenario following the saga pattern
// with operator-triggered retry.
//
// Saga and retry behaviour under test:
//   - Offer reservation succeeds; compensation is registered immediately.
//   - CompleteApplication fails on all 3 retry attempts (retries exhausted).
//   - The workflow records the failure then runs ReleaseOffer compensation
//     explicitly from a disconnected context.
//   - The workflow blocks on FulfilmentRetrySignal, waiting for an operator action.
//   - On receipt of the signal, the workflow re-reserves the offer (idempotent)
//     and re-runs fulfilment with no failure injection.
//   - The workflow completes successfully with StatusCompleted.
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

	// Credit check signal: delivered early so upstream activities complete first.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	// Retry signal: delivered after compensation has run and the workflow is
	// blocking on FulfilmentRetrySignal. 30 s of simulated time is well past the
	// 3-attempt retry exhaustion (1 s + 2 s backoff) and compensation activity.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(FulfilmentRetrySignal, nil)
	}, 30*time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError(), "workflow must complete without error after retry succeeds")

	var result MortgageApplication
	assert.NoError(t, env.GetWorkflowResult(&result))

	assert.Equal(t, StatusCompleted, result.Status)
	assert.Equal(t, "completed", result.CurrentStep)
	assert.NotEmpty(t, result.OfferID, "offerId must be set after successful retry")

	// Build a lookup map and an ordered step list for ordering assertions.
	byKey := make(map[string]TimelineEntry, len(result.Timeline))
	steps := make([]string, len(result.Timeline))
	for i, e := range result.Timeline {
		key := e.Step + "/" + string(e.Status)
		byKey[key] = e
		steps[i] = key
	}

	// Fulfilment failure and compensation must appear.
	fulfilmentFailed, ok := byKey["fulfilment/failed"]
	assert.True(t, ok, "audit trail must include fulfilment/failed after retry exhaustion")
	assert.Equal(t, "OFFER-"+testApplicationID, fulfilmentFailed.Metadata["offerId"])
	assert.Equal(t, "Maximum retry attempts exhausted", fulfilmentFailed.Metadata["reason"])

	compStarted, ok := byKey["compensation/started"]
	assert.True(t, ok, "audit trail must include compensation/started")
	assert.Equal(t, "OFFER-"+testApplicationID, compStarted.Metadata["offerId"])

	compCompleted, ok := byKey["compensation/completed"]
	assert.True(t, ok, "audit trail must include compensation/completed")
	assert.Equal(t, "OFFER-"+testApplicationID, compCompleted.Metadata["offerId"])
	assert.Equal(t, string(StatusCompensated), compCompleted.Metadata["status"])

	// Retry wait must appear after compensation.
	_, hasRetryWait := byKey["fulfilment_retry/waiting"]
	assert.True(t, hasRetryWait, "audit trail must include fulfilment_retry/waiting")

	// Successful fulfilment on retry (byKey holds the last entry for duplicate keys).
	fulfilmentCompleted, ok := byKey["fulfilment/completed"]
	assert.True(t, ok, "audit trail must include fulfilment/completed after retry")
	assert.Equal(t, "OFFER-"+testApplicationID, fulfilmentCompleted.Metadata["offerId"])

	// Compensation must complete before the retry wait in the timeline.
	var compCompletedIdx, retryWaitIdx int
	for i, s := range steps {
		if s == "compensation/completed" {
			compCompletedIdx = i
		}
		if s == "fulfilment_retry/waiting" {
			retryWaitIdx = i
		}
	}
	assert.Less(t, compCompletedIdx, retryWaitIdx, "compensation must complete before retry wait")
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.name, tc.got)
		})
	}
}
