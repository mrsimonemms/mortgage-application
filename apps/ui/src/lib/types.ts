export type ApplicationStatus =
  | 'submitted'
  | 'credit_check_pending'
  | 'offer_reserved'
  | 'completed'
  | 'rejected'
  | 'compensation_required'
  | 'compensated';

export type TimelineStatus = 'started' | 'completed' | 'failed' | 'waiting';

export interface TimelineEntry {
  step: string;
  status: TimelineStatus;
  timestamp: string;
  details?: string;
  metadata?: Record<string, string>;
}

export type SlaStatus = 'within_sla' | 'sla_breached';

// Application-level workflow lifecycle status. The API normalises Temporal's
// raw status names into this enum at the system boundary; the UI never sees
// or compares against Temporal-specific strings.
export type ApplicationWorkflowStatus =
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'terminated'
  | 'timed_out'
  | 'continued_as_new'
  | 'unknown';

export interface MortgageApplication {
  applicationId: string;
  applicantName: string;
  status: ApplicationStatus;
  currentStep: string;
  offerId?: string;
  createdAt: string;
  updatedAt: string;
  timeline: TimelineEntry[];
  pendingDependency?: string;
  pendingSince?: string;
  slaDeadline?: string;
  slaStatus?: SlaStatus;
  slaBreached?: boolean;
  workflowStatus?: ApplicationWorkflowStatus;
}

export interface ScenarioOption {
  name: string;
  description: string;
}

export interface ApplicationListItem {
  applicationId: string;
  applicantName: string;
  scenario?: string;
  workflowStatus: ApplicationWorkflowStatus;
}
