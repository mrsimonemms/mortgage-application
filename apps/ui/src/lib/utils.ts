import type { ApplicationWorkflowStatus } from './types';

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

export function statusLabel(s: string): string {
  return s.replace(/_/g, ' ');
}

// Application-level lifecycle states that mean the workflow has stopped
// running for reasons OTHER than completing naturally. Each stops the
// workflow immediately without giving cleanup logic a chance to run, so
// SLA fields can be left in a transient state that the UI must not
// continue treating as live.
export const NON_RUNNING_TERMINAL_STATUSES = [
  'terminated',
  'cancelled',
  'timed_out',
  'failed',
] as const satisfies readonly ApplicationWorkflowStatus[];

export type NonRunningTerminalStatus =
  (typeof NON_RUNNING_TERMINAL_STATUSES)[number];

export function isNonRunningTerminal(
  status: ApplicationWorkflowStatus | undefined,
): status is NonRunningTerminalStatus {
  return (
    status !== undefined &&
    (NON_RUNNING_TERMINAL_STATUSES as readonly string[]).includes(status)
  );
}

// Human-readable label for any application-level workflow status. Kept
// distinct from business outcomes (approved/rejected/within_sla/sla_breached)
// so the UI never conflates a lifecycle event with an SLA result.
const WORKFLOW_STATUS_LABELS: Record<ApplicationWorkflowStatus, string> = {
  running: 'Running',
  completed: 'Completed',
  failed: 'Failed',
  cancelled: 'Cancelled',
  terminated: 'Terminated',
  timed_out: 'Timed out',
  continued_as_new: 'Continued as new',
  unknown: 'Unknown',
};

export function workflowStatusLabel(status: ApplicationWorkflowStatus): string {
  return WORKFLOW_STATUS_LABELS[status];
}

export function lifecycleLabel(status: NonRunningTerminalStatus): string {
  return WORKFLOW_STATUS_LABELS[status];
}
