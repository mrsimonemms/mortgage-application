import { MortgageScenario } from '../models/mortgage-scenario.type';

export interface MortgageApplicationSubmitted {
  applicationId: string;
  applicantName: string;
  submittedAt: string;
  scenario?: MortgageScenario;
  externalFailureRatePercent?: number;
}
