import { Test, TestingModule } from '@nestjs/testing';

import { WORKFLOW_CLIENT } from '../temporal/temporal.providers';
import { MORTGAGE_SCENARIOS } from './models/mortgage-scenario.type';
import { MortgageController } from './mortgage.controller';
import { MortgageService } from './mortgage.service';

describe('MortgageController', () => {
  let controller: MortgageController;

  const mockWorkflowClient = {
    workflow: { start: jest.fn(), getHandle: jest.fn() },
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [MortgageController],
      providers: [
        MortgageService,
        { provide: WORKFLOW_CLIENT, useValue: mockWorkflowClient },
      ],
    }).compile();

    controller = module.get<MortgageController>(MortgageController);
  });

  describe('getScenarios', () => {
    it('returns an object with a scenarios array', () => {
      const result = controller.getScenarios();
      expect(result).toHaveProperty('scenarios');
      expect(Array.isArray(result.scenarios)).toBe(true);
    });

    it('returns all entries from MORTGAGE_SCENARIOS with name and description', () => {
      const result = controller.getScenarios();
      expect(result.scenarios).toEqual(MORTGAGE_SCENARIOS);
      result.scenarios.forEach((scenario) => {
        expect(scenario).toHaveProperty('name');
        expect(scenario).toHaveProperty('description');
        expect(typeof scenario.name).toBe('string');
        expect(typeof scenario.description).toBe('string');
      });
    });
  });
});
