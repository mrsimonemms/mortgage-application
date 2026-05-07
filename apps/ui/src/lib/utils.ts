import type { ApplicationWorkflowStatus, WorkflowVersion } from './types';

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

// Human-readable label for the workflow version badge. v1/v2 are kept as
// short, lowercase tokens so the badge reads consistently with the existing
// CLI/log naming. The unknown bucket is spelled out so the user sees
// 'Unknown' rather than a placeholder character.
const WORKFLOW_VERSION_LABELS: Record<WorkflowVersion, string> = {
  v1: 'v1',
  v2: 'v2',
  unknown: 'Unknown',
};

export function workflowVersionLabel(version: WorkflowVersion): string {
  return WORKFLOW_VERSION_LABELS[version];
}

// Subtle CSS for the v1/v2/unknown badge. v2 gets a distinct teal so the
// updated workflow slice is identifiable at a glance during the demo, while
// v1 stays a neutral grey to emphasise that legacy executions are unchanged.
export function workflowVersionStyle(version: WorkflowVersion): string {
  switch (version) {
    case 'v2':
      return 'background:#ecfeff;color:#0e7490;border-color:#a5f3fc';
    case 'v1':
      return 'background:#f3f4f6;color:#374151;border-color:#d1d5db';
    default:
      return 'background:#f9fafb;color:#9ca3af;border-color:#e5e7eb';
  }
}
