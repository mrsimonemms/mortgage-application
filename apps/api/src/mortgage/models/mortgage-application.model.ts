import { ApplicationStatus } from './application-status.type.js';
import { TimelineEntry } from './timeline-entry.model.js';

export interface MortgageApplication {
  applicationId: string;
  applicantName: string;
  status: ApplicationStatus;
  currentStep: string;
  offerId?: string;
  createdAt: string;
  updatedAt: string;
  timeline: TimelineEntry[];
}
