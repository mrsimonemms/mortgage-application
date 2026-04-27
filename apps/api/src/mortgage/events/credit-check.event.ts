export type CreditCheckResult = 'approved' | 'rejected';

export interface CreditCheck {
  applicationId: string;
  result: CreditCheckResult;
  reference?: string;
}
