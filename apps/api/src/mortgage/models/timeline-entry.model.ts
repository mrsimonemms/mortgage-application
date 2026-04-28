export type TimelineStatus = 'started' | 'completed' | 'failed' | 'waiting';

export interface TimelineEntry {
  step: string;
  status: TimelineStatus;
  timestamp: string;
  details?: string;
  metadata?: Record<string, string>;
}
