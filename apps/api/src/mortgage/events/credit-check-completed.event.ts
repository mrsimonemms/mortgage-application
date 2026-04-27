export type CreditCheckResult = 'approved' | 'rejected';

export interface CreditCheckCompleted {
  applicationId: string;
  result: CreditCheckResult;
  completedAt: string;
  reference?: string;
}
