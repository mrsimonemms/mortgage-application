import { ConflictException, NotFoundException } from '@nestjs/common';
import { Test, TestingModule } from '@nestjs/testing';
import { WorkflowNotFoundError } from '@temporalio/client';

import { WORKFLOW_CLIENT } from '../temporal/temporal.providers';
import { WorkflowExecutionStatus } from './models/application-workflow-status.type';
import { MortgageService } from './mortgage.service';

// Tests pass the actual Temporal proto enum values to the mocked describe()
// so the normaliser is exercised the same way it is in production. No raw
// status strings appear in this file.
const STATUS_RUNNING = {
  status: { code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_RUNNING },
};

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
    it('returns application state merged with the running workflow status', async () => {
      const mockApp = { applicationId: 'app-123', status: 'submitted' };
      mockHandle.query.mockResolvedValue(mockApp);

      const result = await service.getApplication('app-123');

      expect(mockWorkflowClient.workflow.getHandle).toHaveBeenCalledWith(
        'mortgage-application-app-123',
      );
      expect(mockHandle.query).toHaveBeenCalledWith('getApplication');
      // The API normalises Temporal's proto enum value to the canonical
      // application-level status before returning, so callers never see
      // Temporal-specific constants or naming variants.
      expect(result).toEqual({ ...mockApp, workflowStatus: 'running' });
    });

    it('returns application state merged with the completed workflow status', async () => {
      const mockApp = { applicationId: 'app-123', status: 'completed' };
      mockHandle.query.mockResolvedValue(mockApp);
      mockHandle.describe.mockResolvedValue({
        status: {
          code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_COMPLETED,
        },
      });

      const result = await service.getApplication('app-123');

      expect(result).toEqual({ ...mockApp, workflowStatus: 'completed' });
    });

    // Each Temporal proto enum value must normalise to the exact
    // application-level status the UI expects. Tests pass the enum value
    // itself (the same numeric code Temporal returns on `desc.status.code`)
    // rather than relying on the SDK's string name, so the normaliser is
    // exercised against the same input shape it sees in production.
    it.each([
      {
        code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_TERMINATED,
        expected: 'terminated',
      },
      {
        // Temporal's proto enum is the American spelling. The mapping
        // intentionally produces the British `cancelled` for consistency
        // with the rest of the application's vocabulary.
        code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_CANCELED,
        expected: 'cancelled',
      },
      {
        code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_TIMED_OUT,
        expected: 'timed_out',
      },
      {
        code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_FAILED,
        expected: 'failed',
      },
      {
        code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW,
        expected: 'continued_as_new',
      },
    ])(
      'normalises Temporal status code=$code to application-level status $expected',
      async ({ code, expected }) => {
        const mockApp = {
          applicationId: 'app-123',
          status: 'credit_check_pending',
          pendingDependency: 'credit_check',
        };
        mockHandle.query.mockResolvedValue(mockApp);
        mockHandle.describe.mockResolvedValue({ status: { code } });

        const result = await service.getApplication('app-123');

        expect(result.workflowStatus).toBe(expected);
        // Mid-flight query data is preserved verbatim; only the lifecycle
        // hint is added on top so the UI can suppress live SLA visuals.
        expect(result).toMatchObject(mockApp);
      },
    );

    // Statuses we do not explicitly map (the `_UNSPECIFIED` / `_PAUSED`
    // proto values, undefined, and any future Temporal additions) collapse
    // to the single `unknown` bucket so the UI never has to handle a value
    // it does not recognise. The 999 case simulates a future Temporal enum
    // value that this codebase has not yet been updated to handle.
    it.each([
      { code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED },
      { code: WorkflowExecutionStatus.WORKFLOW_EXECUTION_STATUS_PAUSED },
      { code: 999 },
      { code: undefined },
    ])('normalises Temporal status code=$code to unknown', async ({ code }) => {
      const mockApp = { applicationId: 'app-123', status: 'submitted' };
      mockHandle.query.mockResolvedValue(mockApp);
      mockHandle.describe.mockResolvedValue({ status: { code } });

      const result = await service.getApplication('app-123');

      expect(result.workflowStatus).toBe('unknown');
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
