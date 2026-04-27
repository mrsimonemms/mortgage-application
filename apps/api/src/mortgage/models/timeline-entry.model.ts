export type TimelineStatus = 'started' | 'completed' | 'failed';

export interface TimelineEntry {
  step: string;
  status: TimelineStatus;
  timestamp: string;
  details?: string;
}
