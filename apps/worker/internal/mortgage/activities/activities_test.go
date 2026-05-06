package activities

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func init() {
	DisableActivityDelaysForTests()
}

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

func TestMaybeFailExternalDependency(t *testing.T) {
	t.Run("never fails when rate is zero", func(t *testing.T) {
		for range 200 {
			assert.NoError(t, maybeFailExternalDependency("TestActivity", 0))
		}
	})

	t.Run("never fails when rate is negative", func(t *testing.T) {
		for range 200 {
			assert.NoError(t, maybeFailExternalDependency("TestActivity", -10))
		}
	})

	t.Run("error message includes activity name", func(t *testing.T) {
		// Override randIntn so it always returns 0, guaranteeing failure.
		orig := randIntn
		randIntn = func(_ int) int { return 0 }
		defer func() { randIntn = orig }()

		err := maybeFailExternalDependency("MyActivity", MaxExternalFailureRatePercent)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "MyActivity")
		}
	})

	t.Run("values above max are clamped rather than causing a panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = maybeFailExternalDependency("TestActivity", 999)
		})
	})
}

func TestPropertyValuation(t *testing.T) {
	t.Run("returns a deterministic valuation id", func(t *testing.T) {
		env := newTestEnv(t)

		val, err := env.ExecuteActivity(Activities{}.PropertyValuation, PropertyValuationInput{
			ApplicationID: "APP-001",
		})

		assert.NoError(t, err)
		var result PropertyValuationResult
		assert.NoError(t, val.Get(&result))
		assert.Equal(t, "APP-001", result.ApplicationID)
		assert.Equal(t, "VAL-APP-001", result.ValuationID)
		assert.False(t, result.ValuedAt.IsZero())
	})

	t.Run("rejects empty application id", func(t *testing.T) {
		env := newTestEnv(t)

		_, err := env.ExecuteActivity(Activities{}.PropertyValuation, PropertyValuationInput{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "applicationId")
	})

	// Property valuation participates in the same failure-injection pattern as
	// the other external activities. With randIntn forced to 0 the maximum
	// failure rate guarantees a retryable simulated failure so Temporal drives
	// the retries automatically.
	t.Run("respects external failure injection with a retryable error", func(t *testing.T) {
		orig := randIntn
		randIntn = func(_ int) int { return 0 }
		defer func() { randIntn = orig }()

		env := newTestEnv(t)

		_, err := env.ExecuteActivity(Activities{}.PropertyValuation, PropertyValuationInput{
			ApplicationID:              "APP-001",
			ExternalFailureRatePercent: MaxExternalFailureRatePercent,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PropertyValuation")
	})
}

func TestSendNotification(t *testing.T) {
	t.Run("dispatches notification with application id and status", func(t *testing.T) {
		env := newTestEnv(t)

		val, err := env.ExecuteActivity(Activities{}.SendNotification, SendNotificationInput{
			ApplicationID: "APP-001",
			Status:        "approved",
		})

		assert.NoError(t, err)
		var result SendNotificationResult
		assert.NoError(t, val.Get(&result))
		assert.Equal(t, "APP-001", result.ApplicationID)
		assert.Equal(t, "approved", result.Status)
		assert.False(t, result.DeliveredAt.IsZero())
	})

	t.Run("rejects empty application id with non-retryable error", func(t *testing.T) {
		env := newTestEnv(t)

		_, err := env.ExecuteActivity(Activities{}.SendNotification, SendNotificationInput{
			Status: "approved",
		})

		assert.Error(t, err)
		var appErr *temporal.ApplicationError
		errors.As(err, &appErr)
		if assert.NotNil(t, appErr, "error must be a temporal.ApplicationError") {
			assert.True(t, appErr.NonRetryable(),
				"missing applicationId is a wiring bug, not a transient failure")
		}
		assert.Contains(t, err.Error(), "applicationId")
	})

	t.Run("rejects empty status with non-retryable error", func(t *testing.T) {
		env := newTestEnv(t)

		_, err := env.ExecuteActivity(Activities{}.SendNotification, SendNotificationInput{
			ApplicationID: "APP-001",
		})

		assert.Error(t, err)
		var appErr *temporal.ApplicationError
		errors.As(err, &appErr)
		if assert.NotNil(t, appErr) {
			assert.True(t, appErr.NonRetryable())
		}
		assert.Contains(t, err.Error(), "status")
	})

	// SendNotification participates in the same failure-injection pattern as
	// the other external activities. With randIntn forced to always return 0
	// the maximum failure rate guarantees a retryable simulated failure.
	t.Run("respects external failure injection with a retryable error", func(t *testing.T) {
		orig := randIntn
		randIntn = func(_ int) int { return 0 }
		defer func() { randIntn = orig }()

		env := newTestEnv(t)

		_, err := env.ExecuteActivity(Activities{}.SendNotification, SendNotificationInput{
			ApplicationID:              "APP-001",
			Status:                     "approved",
			ExternalFailureRatePercent: MaxExternalFailureRatePercent,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SendNotification")

		// The injected error must be retryable so Temporal drives the backoff
		// automatically, matching the behaviour of every other activity.
		var appErr *temporal.ApplicationError
		errors.As(err, &appErr)
		if assert.NotNil(t, appErr) {
			assert.False(t, appErr.NonRetryable(),
				"external failure injection must produce a retryable error")
		}
	})
}

func TestReleaseOffer(t *testing.T) {
	t.Run("releases an offer successfully", func(t *testing.T) {
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
	})

	// ReleaseOffer is idempotent: repeated calls for the same offerId succeed without
	// error. Temporal may retry the compensation activity, and each retry must produce
	// the same logical outcome without creating duplicate side effects.
	t.Run("is idempotent: repeated calls for the same offerId succeed", func(t *testing.T) {
		env := newTestEnv(t)

		val1, err := env.ExecuteActivity(Activities{}.ReleaseOffer, ReleaseOfferInput{
			ApplicationID: "APP-001",
			OfferID:       "OFFER-APP-001",
		})
		assert.NoError(t, err)
		var r1 ReleaseOfferResult
		assert.NoError(t, val1.Get(&r1))
		assert.Equal(t, "APP-001", r1.ApplicationID)

		val2, err := env.ExecuteActivity(Activities{}.ReleaseOffer, ReleaseOfferInput{
			ApplicationID: "APP-001",
			OfferID:       "OFFER-APP-001",
		})
		assert.NoError(t, err)
		var r2 ReleaseOfferResult
		assert.NoError(t, val2.Get(&r2))
		assert.Equal(t, "APP-001", r2.ApplicationID)
		assert.False(t, r2.ReleasedAt.IsZero())
	})
}
