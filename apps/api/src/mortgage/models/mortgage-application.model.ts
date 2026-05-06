import { ApplicationStatus } from './application-status.type.js';
import { ApplicationWorkflowStatus } from './application-workflow-status.type.js';
import { TimelineEntry } from './timeline-entry.model.js';

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
  // Application-level workflow lifecycle status. The API normalises Temporal's
  // raw status names into this enum at the boundary so the UI can stop showing
  // live SLA visuals when the workflow is no longer running, without ever
  // depending on Temporal-specific naming.
  workflowStatus?: ApplicationWorkflowStatus;
}
