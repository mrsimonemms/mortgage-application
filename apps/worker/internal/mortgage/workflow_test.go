package mortgage

import (
	"testing"
	"time"

	"github.com/mrsimonemms/mortgage-application/mortgage-application/apps/worker/internal/mortgage/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestMortgageApplicationWorkflow_HappyPath runs the full workflow through the
// Temporal test environment.
//
// The credit check signal is registered before execution begins. Temporal buffers
// signals, so the signal is held until the workflow reaches the durable wait point
// (workflow.GetSignalChannel(...).Receive(...)), at which point it is consumed and
// execution resumes. This confirms that the workflow correctly pauses and that the
// signal mechanism drives the transition.
func TestMortgageApplicationWorkflow_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(MortgageApplicationWorkflow)
	env.RegisterActivity(&activities.Activities{})

	input := MortgageApplicationSubmitted{
		ApplicationID: testApplicationID,
		ApplicantName: testApplicantName,
		SubmittedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Deliver the credit result at a positive delay so the workflow has run its
	// activities first. The test environment advances simulated time automatically
	// when the workflow blocks, so the callback fires as soon as the workflow is
	// durably waiting for the signal.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CreditCheckCompletedSignal, CreditCheckCompleted{
			ApplicationID: testApplicationID,
			Result:        CreditCheckApproved,
			CompletedAt:   time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		})
	}, time.Second)

	env.ExecuteWorkflow(MortgageApplicationWorkflow, input)

	// Fatal preconditions: if the workflow did not complete cleanly we cannot
	// safely inspect the result.
	require.True(t, env.IsWorkflowCompleted(), "workflow should have completed")
	require.NoError(t, env.GetWorkflowError())

	var result MortgageApplication
	require.NoError(t, env.GetWorkflowResult(&result))

	// Final application state.
	assert.Equal(t, StatusCompleted, result.Status)
	assert.Equal(t, "completed", result.CurrentStep)
	assert.Equal(t, testApplicationID, result.ApplicationID)
	assert.Equal(t, testApplicantName, result.ApplicantName)
	assert.NotEmpty(t, result.OfferID)

	// Timeline must contain all expected steps in order.
	steps := make([]string, len(result.Timeline))
	for i, e := range result.Timeline {
		steps[i] = e.Step + "/" + string(e.Status)
	}

	assert.Equal(t, []string{
		"submitted/completed",
		"intake/started",
		"intake/completed",
		"credit_check/started",
		"credit_check/completed",
		"offer_reservation/started",
		"offer_reservation/completed",
		"fulfilment/started",
		"fulfilment/completed",
	}, steps)
}
