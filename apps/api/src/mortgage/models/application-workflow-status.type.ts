import * as proto from '@temporalio/proto';

// Canonical, application-level workflow status. The API normalises Temporal's
// SDK/protobuf-managed enum values into this shape at the system boundary so
// downstream consumers (API responses, UI) never see Temporal-specific
// constants and never need to handle SDK-version-specific naming variants.
export type ApplicationWorkflowStatus =
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'terminated'
  | 'timed_out'
  | 'continued_as_new'
  | 'unknown';

export const APPLICATION_WORKFLOW_STATUSES: ApplicationWorkflowStatus[] = [
  'running',
  'completed',
  'failed',
  'cancelled',
  'terminated',
  'timed_out',
  'continued_as_new',
  'unknown',
];

// Re-export the Temporal proto enum under a stable local alias so call sites
// (and tests) can refer to enum members without depending on the long
// `proto.temporal.api.enums.v1.WorkflowExecutionStatus` qualifier directly.
export const WorkflowExecutionStatus =
  proto.temporal.api.enums.v1.WorkflowExecutionStatus;
export type WorkflowExecutionStatus =
  proto.temporal.api.enums.v1.WorkflowExecutionStatus;

// normaliseWorkflowStatus maps a Temporal-managed protobuf enum value onto
// the application-level enum. It accepts the same numeric enum value Temporal
// exposes on `desc.status.code` and `info.status.code`, so the API never has
// to depend on the SDK's spelling of the equivalent string name (which has
// historically diverged between SDKs and versions, e.g. CANCELED/CANCELLED).
//
// Anything we do not explicitly handle (UNSPECIFIED, PAUSED, undefined,
// future additions) collapses to `unknown` so the UI only ever has to deal
// with the values listed in ApplicationWorkflowStatus.
export function normaliseWorkflowStatus(
  status: WorkflowExecutionStatus | undefined,
): ApplicationWorkflowStatus {
  switch (status) {
    case WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_RUNNING:
      return 'running';
    case WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_COMPLETED:
      return 'completed';
    case WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_FAILED:
      return 'failed';
    case WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_CANCELED:
      return 'cancelled';
    case WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_TERMINATED:
      return 'terminated';
    case WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
      return 'timed_out';
    case WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
      return 'continued_as_new';
    default:
      return 'unknown';
  }
}
