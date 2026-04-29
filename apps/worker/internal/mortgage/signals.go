package mortgage

const (
	// CreditCheckCompletedSignal is the signal name used to deliver an async credit result.
	CreditCheckCompletedSignal = "credit-check-completed"

	// FulfilmentRetrySignal is sent by an operator to re-attempt fulfilment after
	// compensation has run and the reserved offer has been released.
	FulfilmentRetrySignal = "retry-fulfilment"

	// QueryApplication is the query name used to read current workflow state.
	QueryApplication = "getApplication"
)
