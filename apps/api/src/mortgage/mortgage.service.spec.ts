import { ConflictException, NotFoundException } from '@nestjs/common';
import { Test, TestingModule } from '@nestjs/testing';
import { WorkflowNotFoundError } from '@temporalio/client';

import { WORKFLOW_CLIENT } from '../temporal/temporal.providers';
import { MortgageService } from './mortgage.service';

// 1 = WORKFLOW_EXECUTION_STATUS_RUNNING (temporal.api.enums.v1.WorkflowExecutionStatus)
const STATUS_RUNNING = { status: { code: 1 } };

describe('MortgageService', () => {
  let service: MortgageService;

  const mockHandle = {
    query: jest.fn(),
    signal: jest.fn(),
    describe: jest.fn(),
  };

  const mockWorkflowClient = {
    workflow: {
      start: jest.fn(),
      getHandle: jest.fn(),
    },
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        MortgageService,
        { provide: WORKFLOW_CLIENT, useValue: mockWorkflowClient },
      ],
    }).compile();

    service = module.get<MortgageService>(MortgageService);

    jest.clearAllMocks();
    mockWorkflowClient.workflow.start.mockResolvedValue(mockHandle);
    mockWorkflowClient.workflow.getHandle.mockReturnValue(mockHandle);
    // Default: workflow is running. Override per test where a different state is needed.
    mockHandle.describe.mockResolvedValue(STATUS_RUNNING);
  });

  describe('startApplication', () => {
    it('starts the workflow with correct type, workflow ID, and task queue', async () => {
      mockHandle.describe.mockRejectedValue(new Error('not found'));

      const result = await service.startApplication('app-123', 'John Smith');

      expect(mockWorkflowClient.workflow.start).toHaveBeenCalledWith(
        'MortgageApplicationWorkflow',
        expect.objectContaining({
          taskQueue: 'mortgage-application',
          workflowId: 'mortgage-application-app-123',
          args: [
            expect.objectContaining({
              applicationId: 'app-123',
              applicantName: 'John Smith',
            }),
          ],
        }),
      );
      expect(result).toEqual({
        workflowId: 'mortgage-application-app-123',
        applicationId: 'app-123',
      });
    });

    it('sends happy_path scenario when no scenario is specified', async () => {
      mockHandle.describe.mockRejectedValue(new Error('not found'));

      await service.startApplication('app-123', 'John Smith');

      expect(mockWorkflowClient.workflow.start).toHaveBeenCalledWith(
        'MortgageApplicationWorkflow',
        expect.objectContaining({
          args: [expect.objectContaining({ scenario: 'happy_path' })],
        }),
      );
    });

    it('sends happy_path scenario when happy_path is specified', async () => {
      mockHandle.describe.mockRejectedValue(new Error('not found'));

      await service.startApplication('app-123', 'John Smith', 'happy_path');

      expect(mockWorkflowClient.workflow.start).toHaveBeenCalledWith(
        'MortgageApplicationWorkflow',
        expect.objectContaining({
          args: [expect.objectContaining({ scenario: 'happy_path' })],
        }),
      );
    });

    it('sends fail_after_offer_reservation scenario when specified', async () => {
      mockHandle.describe.mockRejectedValue(new Error('not found'));

      await service.startApplication(
        'app-123',
        'John Smith',
        'fail_after_offer_reservation',
      );

      expect(mockWorkflowClient.workflow.start).toHaveBeenCalledWith(
        'MortgageApplicationWorkflow',
        expect.objectContaining({
          args: [
            expect.objectContaining({
              scenario: 'fail_after_offer_reservation',
            }),
          ],
        }),
      );
    });

    it('sends fail_and_compensate_after_offer_reservation scenario when specified', async () => {
      mockHandle.describe.mockRejectedValue(new Error('not found'));

      await service.startApplication(
        'app-123',
        'John Smith',
        'fail_and_compensate_after_offer_reservation',
      );

      expect(mockWorkflowClient.workflow.start).toHaveBeenCalledWith(
        'MortgageApplicationWorkflow',
        expect.objectContaining({
          args: [
            expect.objectContaining({
              scenario: 'fail_and_compensate_after_offer_reservation',
            }),
          ],
        }),
      );
    });

    it('throws ConflictException when a workflow is already running', async () => {
      // default: mockHandle.describe resolves with STATUS_RUNNING
      await expect(
        service.startApplication('app-123', 'John Smith'),
      ).rejects.toThrow(ConflictException);
    });

    it('throws ConflictException when a workflow already completed', async () => {
      mockHandle.describe.mockResolvedValue({ status: { code: 2 } }); // COMPLETED

      await expect(
        service.startApplication('app-123', 'John Smith'),
      ).rejects.toThrow(ConflictException);
    });
  });

  describe('getApplication', () => {
    it('returns application state for a running workflow', async () => {
      const mockApp = { applicationId: 'app-123', status: 'submitted' };
      mockHandle.query.mockResolvedValue(mockApp);

      const result = await service.getApplication('app-123');

      expect(mockWorkflowClient.workflow.getHandle).toHaveBeenCalledWith(
        'mortgage-application-app-123',
      );
      expect(mockHandle.query).toHaveBeenCalledWith('getApplication');
      expect(result).toEqual(mockApp);
    });

    it('returns application state for a completed workflow', async () => {
      const mockApp = { applicationId: 'app-123', status: 'completed' };
      mockHandle.query.mockResolvedValue(mockApp);

      const result = await service.getApplication('app-123');

      expect(result).toEqual(mockApp);
    });

    it('throws NotFoundException when the workflow does not exist', async () => {
      mockHandle.query.mockRejectedValue(
        new WorkflowNotFoundError(
          'workflow not found',
          'mortgage-application-app-123',
          undefined,
        ),
      );

      await expect(service.getApplication('app-123')).rejects.toThrow(
        NotFoundException,
      );
    });

    it('propagates unexpected errors from the query', async () => {
      mockHandle.query.mockRejectedValue(
        new Error('unexpected temporal error'),
      );

      await expect(service.getApplication('app-123')).rejects.toThrow(
        'unexpected temporal error',
      );
    });
  });

  describe('retryFulfilment', () => {
    it('sends retry-fulfilment signal to the running workflow', async () => {
      await service.retryFulfilment('app-123');

      expect(mockWorkflowClient.workflow.getHandle).toHaveBeenCalledWith(
        'mortgage-application-app-123',
      );
      expect(mockHandle.signal).toHaveBeenCalledWith('retry-fulfilment');
    });

    it('throws NotFoundException when the workflow is not running', async () => {
      mockHandle.describe.mockRejectedValue(new Error('not found'));

      await expect(service.retryFulfilment('app-123')).rejects.toThrow(
        NotFoundException,
      );
    });
  });

  describe('completeCreditCheck', () => {
    it('sends credit-check-completed signal with correct payload', async () => {
      await service.completeCreditCheck('app-123', 'approved', 'REF-001');

      expect(mockWorkflowClient.workflow.getHandle).toHaveBeenCalledWith(
        'mortgage-application-app-123',
      );
      expect(mockHandle.signal).toHaveBeenCalledWith(
        'credit-check-completed',
        expect.objectContaining({
          applicationId: 'app-123',
          result: 'approved',
          reference: 'REF-001',
        }),
      );
    });

    it('sends the signal without reference when omitted', async () => {
      await service.completeCreditCheck('app-123', 'rejected');

      expect(mockHandle.signal).toHaveBeenCalledWith(
        'credit-check-completed',
        expect.not.objectContaining({
          reference: expect.anything() as unknown,
        }),
      );
      expect(mockHandle.signal).toHaveBeenCalledWith(
        'credit-check-completed',
        expect.objectContaining({
          applicationId: 'app-123',
          result: 'rejected',
        }),
      );
    });

    it('throws NotFoundException when the workflow is not running', async () => {
      mockHandle.describe.mockRejectedValue(new Error('not found'));

      await expect(
        service.completeCreditCheck('app-123', 'approved'),
      ).rejects.toThrow(NotFoundException);
    });
  });
});
