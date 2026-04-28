export interface MortgageScenarioOption {
  name: string;
  description: string;
}

export const MORTGAGE_SCENARIOS: MortgageScenarioOption[] = [
  {
    name: 'happy_path',
    description: 'Full successful mortgage workflow.',
  },
  {
    name: 'fail_after_offer_reservation',
    description:
      'Fulfilment fails on the first four attempts. Temporal retries automatically and the workflow completes on the fifth attempt.',
  },
  {
    name: 'fail_and_compensate_after_offer_reservation',
    description:
      'Fulfilment fails on all retry attempts. The workflow compensates by releasing the reserved offer and ends in a compensated terminal state.',
  },
];

export type MortgageScenario = (typeof MORTGAGE_SCENARIOS)[number]['name'];
