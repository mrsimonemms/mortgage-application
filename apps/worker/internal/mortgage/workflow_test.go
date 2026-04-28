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
