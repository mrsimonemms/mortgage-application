package activities

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func newTestEnv(t *testing.T) *testsuite.TestActivityEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(&Activities{})
	return env
}

func TestIntake(t *testing.T) {
	t.Run("succeeds with valid input", func(t *testing.T) {
		env := newTestEnv(t)

		val, err := env.ExecuteActivity(Activities{}.Intake, IntakeInput{
			ApplicationID: "APP-001",
			ApplicantName: "Jane Smith",
		})

		assert.NoError(t, err)
		var result IntakeResult
		assert.NoError(t, val.Get(&result))
		assert.Equal(t, "APP-001", result.ApplicationID)
		assert.False(t, result.ReceivedAt.IsZero())
	})

	t.Run("fails when applicationId is empty", func(t *testing.T) {
		env := newTestEnv(t)

		_, err := env.ExecuteActivity(Activities{}.Intake, IntakeInput{
			ApplicantName: "Jane Smith",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "applicationId")
	})

	t.Run("fails when applicantName is empty", func(t *testing.T) {
		env := newTestEnv(t)

		_, err := env.ExecuteActivity(Activities{}.Intake, IntakeInput{
			ApplicationID: "APP-001",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "applicantName")
	})
}

func TestRequestCreditCheck(t *testing.T) {
	env := newTestEnv(t)

	val, err := env.ExecuteActivity(Activities{}.RequestCreditCheck, CreditCheckInput{
		ApplicationID: "APP-001",
	})

	assert.NoError(t, err)
	var result CreditCheckRequestResult
	assert.NoError(t, val.Get(&result))
	assert.Equal(t, "APP-001", result.ApplicationID)
	assert.True(t, strings.HasPrefix(result.Reference, "CREDIT-REQ-"), "reference should have expected prefix")
}

func TestReserveOffer(t *testing.T) {
	t.Run("returns a stable offer ID", func(t *testing.T) {
		env := newTestEnv(t)

		val, err := env.ExecuteActivity(Activities{}.ReserveOffer, ReserveOfferInput{ApplicationID: "APP-001"})

		assert.NoError(t, err)
		var result ReserveOfferResult
		assert.NoError(t, val.Get(&result))
		assert.Equal(t, "APP-001", result.ApplicationID)
		assert.NotEmpty(t, result.OfferID)
		assert.False(t, result.ReservedAt.IsZero())
	})

	t.Run("is idempotent: same application returns same offer ID", func(t *testing.T) {
		env := newTestEnv(t)

		val1, err := env.ExecuteActivity(Activities{}.ReserveOffer, ReserveOfferInput{ApplicationID: "APP-001"})
		if !assert.NoError(t, err) {
			return
		}
		var r1 ReserveOfferResult
		if !assert.NoError(t, val1.Get(&r1)) {
			return
		}

		val2, err := env.ExecuteActivity(Activities{}.ReserveOffer, ReserveOfferInput{ApplicationID: "APP-001"})
		if !assert.NoError(t, err) {
			return
		}
		var r2 ReserveOfferResult
		if !assert.NoError(t, val2.Get(&r2)) {
			return
		}

		assert.Equal(t, r1.OfferID, r2.OfferID)
	})

	t.Run("returns different offer IDs for different applications", func(t *testing.T) {
		env := newTestEnv(t)

		val1, err := env.ExecuteActivity(Activities{}.ReserveOffer, ReserveOfferInput{ApplicationID: "APP-001"})
		if !assert.NoError(t, err) {
			return
		}
		var r1 ReserveOfferResult
		if !assert.NoError(t, val1.Get(&r1)) {
			return
		}

		val2, err := env.ExecuteActivity(Activities{}.ReserveOffer, ReserveOfferInput{ApplicationID: "APP-002"})
		if !assert.NoError(t, err) {
			return
		}
		var r2 ReserveOfferResult
		if !assert.NoError(t, val2.Get(&r2)) {
			return
		}

		assert.NotEqual(t, r1.OfferID, r2.OfferID)
	})
}

func TestCompleteApplication(t *testing.T) {
	t.Run("succeeds on the happy path", func(t *testing.T) {
		env := newTestEnv(t)

		val, err := env.ExecuteActivity(Activities{}.CompleteApplication, CompleteApplicationInput{
			ApplicationID: "APP-001",
			OfferID:       "OFFER-APP-001",
		})

		assert.NoError(t, err)
		var result CompleteApplicationResult
		assert.NoError(t, val.Get(&result))
		assert.Equal(t, "APP-001", result.ApplicationID)
		assert.False(t, result.CompletedAt.IsZero())
	})

	t.Run("fails with a retryable error on early attempts when SimulateFailure is set", func(t *testing.T) {
		env := newTestEnv(t)

		_, err := env.ExecuteActivity(Activities{}.CompleteApplication, CompleteApplicationInput{
			ApplicationID:   "APP-001",
			OfferID:         "OFFER-APP-001",
			SimulateFailure: true,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "completion failure injected for demo")

		// The error must be retryable so Temporal drives the backoff automatically.
		var appErr *temporal.ApplicationError
		errors.As(err, &appErr)
		assert.NotNil(t, appErr, "error must be a temporal.ApplicationError")
		assert.False(t, appErr.NonRetryable(), "error must be retryable so Temporal retries the activity")
	})
}

func TestReleaseOffer(t *testing.T) {
	env := newTestEnv(t)

	val, err := env.ExecuteActivity(Activities{}.ReleaseOffer, ReleaseOfferInput{
		ApplicationID: "APP-001",
		OfferID:       "OFFER-APP-001",
	})

	assert.NoError(t, err)
	var result ReleaseOfferResult
	assert.NoError(t, val.Get(&result))
	assert.Equal(t, "APP-001", result.ApplicationID)
	assert.False(t, result.ReleasedAt.IsZero())
}
