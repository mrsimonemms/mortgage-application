package mortgage

const (
	// CreditCheckCompletedSignal is the signal name used to deliver an async credit result.
	CreditCheckCompletedSignal = "credit-check-completed"

	// RetryCreditCheckSignal is the signal name used by an operator to request a
	// credit check retry. The workflow re-requests the credit check and waits again.
	RetryCreditCheckSignal = "retry-credit-check"

	// PropertyValuationSubmittedSignal is the signal name used to deliver the
	// operator-supplied property value to the v2 workflow. The v2 workflow
	// blocks on this signal between credit approval and offer reservation,
	// passes the submitted value into the property valuation activity and
	// then continues. v1 does not register this signal.
	PropertyValuationSubmittedSignal = "property-valuation-submitted"

	// QueryApplication is the query name used to read current workflow state.
	QueryApplication = "getApplication"
)
