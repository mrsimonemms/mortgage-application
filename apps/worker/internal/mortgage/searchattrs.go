package mortgage

import "go.temporal.io/sdk/temporal"

// Custom search attributes for the mortgage workflow.
// These four attributes capture non-PII business status and execution position,
// enabling demo queries such as filtering by status, step, or waiting state.
var (
	saApplicationStatus      = temporal.NewSearchAttributeKeyKeyword("ApplicationStatus")
	saCurrentStep            = temporal.NewSearchAttributeKeyKeyword("CurrentStep")
	saHasOffer               = temporal.NewSearchAttributeKeyBool("HasOffer")
	saAwaitingExternalSignal = temporal.NewSearchAttributeKeyBool("AwaitingExternalSignal")
)
