package mortgage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// testApplicationID and testApplicantName are shared across test files in this package.
const (
	testApplicationID = "APP-001"
	testApplicantName = "Jane Smith"
)

func TestApplicationStatus_Values(t *testing.T) {
	cases := []struct {
		status   ApplicationStatus
		expected string
	}{
		{StatusSubmitted, "submitted"},
		{StatusCreditCheckPending, "credit_check_pending"},
		{StatusOfferReserved, "offer_reserved"},
		{StatusCompleted, "completed"},
		{StatusRejected, "rejected"},
	}

	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.status))
		})
	}
}

func TestTimelineStatus_Values(t *testing.T) {
	cases := []struct {
		status   TimelineStatus
		expected string
	}{
		{TimelineStarted, "started"},
		{TimelineCompleted, "completed"},
		{TimelineFailed, "failed"},
	}

	for _, tc := range cases {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.status))
		})
	}
}

func TestMortgageApplication_JSONFields(t *testing.T) {
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("all fields present", func(t *testing.T) {
		app := MortgageApplication{
			ApplicationID: testApplicationID,
			ApplicantName: testApplicantName,
			Status:        StatusSubmitted,
			CurrentStep:   string(StatusSubmitted),
			OfferID:       "OFFER-APP-001",
			CreatedAt:     ts,
			UpdatedAt:     ts,
			Timeline: []TimelineEntry{
				{Step: "submitted", Status: TimelineCompleted, Timestamp: ts},
			},
		}

		data, err := json.Marshal(app)
		assert.NoError(t, err)
		var out map[string]any
		assert.NoError(t, json.Unmarshal(data, &out))
		for _, field := range []string{"applicationId", "applicantName", "status", "currentStep", "offerId", "createdAt", "updatedAt", "timeline"} {
			assert.Contains(t, out, field)
		}
	})

	t.Run("offerId omitted when empty", func(t *testing.T) {
		app := MortgageApplication{
			ApplicationID: testApplicationID,
			ApplicantName: testApplicantName,
			Status:        StatusSubmitted,
			CurrentStep:   string(StatusSubmitted),
			CreatedAt:     ts,
			UpdatedAt:     ts,
			Timeline:      []TimelineEntry{},
		}

		data, err := json.Marshal(app)
		assert.NoError(t, err)
		var out map[string]any
		assert.NoError(t, json.Unmarshal(data, &out))
		assert.NotContains(t, out, "offerId")
	})
}

func TestTimelineEntry_JSONFields(t *testing.T) {
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("all fields present", func(t *testing.T) {
		entry := TimelineEntry{
			Step:      "intake",
			Status:    TimelineCompleted,
			Timestamp: ts,
			Details:   "Application received",
		}

		data, err := json.Marshal(entry)
		assert.NoError(t, err)
		var out map[string]any
		assert.NoError(t, json.Unmarshal(data, &out))
		for _, field := range []string{"step", "status", "timestamp", "details"} {
			assert.Contains(t, out, field)
		}
	})

	t.Run("details omitted when empty", func(t *testing.T) {
		entry := TimelineEntry{
			Step:      "intake",
			Status:    TimelineStarted,
			Timestamp: ts,
		}

		data, err := json.Marshal(entry)
		assert.NoError(t, err)
		var out map[string]any
		assert.NoError(t, json.Unmarshal(data, &out))
		assert.NotContains(t, out, "details")
	})
}
