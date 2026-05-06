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
}

export interface ScenarioOption {
  name: string;
  description: string;
}

export type WorkflowExecutionStatusName =
  | 'UNSPECIFIED'
  | 'RUNNING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TERMINATED'
  | 'CONTINUED_AS_NEW'
  | 'TIMED_OUT'
  | 'PAUSED'
  | 'UNKNOWN';

export interface ApplicationListItem {
  applicationId: string;
  applicantName: string;
  scenario?: string;
  workflowStatus: WorkflowExecutionStatusName;
}
