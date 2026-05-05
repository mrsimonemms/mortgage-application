package mortgage

const (
	// CreditCheckCompletedSignal is the signal name used to deliver an async credit result.
	CreditCheckCompletedSignal = "credit-check-completed"

	// RetryCreditCheckSignal is the signal name used by an operator to request a
	// credit check retry. The workflow re-requests the credit check and waits again.
	RetryCreditCheckSignal = "retry-credit-check"

	// QueryApplication is the query name used to read current workflow state.
	QueryApplication = "getApplication"
)
