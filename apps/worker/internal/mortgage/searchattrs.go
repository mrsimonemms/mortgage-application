package mortgage

import "go.temporal.io/sdk/temporal"

// Custom search attributes for the mortgage workflow.
// These attributes capture non-PII business status and execution position,
// enabling demo queries such as filtering by status, step, waiting state or
// SLA outcome. WithinSLA is only set once the SLA outcome of an async wait has
// been finalised; workflows that have not yet finished an SLA-bounded wait do
// not have the attribute populated.
var (
	saApplicationStatus      = temporal.NewSearchAttributeKeyKeyword("ApplicationStatus")
	saCurrentStep            = temporal.NewSearchAttributeKeyKeyword("CurrentStep")
	saHasOffer               = temporal.NewSearchAttributeKeyBool("HasOffer")
	saAwaitingExternalSignal = temporal.NewSearchAttributeKeyBool("AwaitingExternalSignal")
	saWithinSLA              = temporal.NewSearchAttributeKeyBool("WithinSLA")
)
