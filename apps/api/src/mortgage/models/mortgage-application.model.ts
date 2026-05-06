import { ApplicationStatus } from './application-status.type.js';
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
}
