export type MortgageScenario =
  | 'happy_path'
  | 'fail_after_offer_reservation'
  | 'fail_and_compensate_after_offer_reservation';

export const MORTGAGE_SCENARIOS: MortgageScenario[] = [
  'happy_path',
  'fail_after_offer_reservation',
  'fail_and_compensate_after_offer_reservation',
];
