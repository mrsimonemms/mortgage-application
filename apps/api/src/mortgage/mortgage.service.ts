import {
  ConflictException,
  Inject,
  Injectable,
  Logger,
  NotFoundException,
} from '@nestjs/common';
import { Client, WorkflowNotFoundError } from '@temporalio/client';
import * as proto from '@temporalio/proto';

import { WORKFLOW_CLIENT } from '../temporal/temporal.providers';
import { CreditCheckResult } from './events/credit-check.event';
import { MortgageApplication } from './models/mortgage-application.model';
import { MortgageScenario } from './models/mortgage-scenario.type';

// The workflow type name is the short function name that the Temporal Go SDK
// derives from runtime.FuncForPC. It must match the Go worker registration.
const WORKFLOW_TYPE = 'MortgageApplicationWorkflow';
const TASK_QUEUE = 'mortgage-application';
const SIGNAL_CREDIT_CHECK_COMPLETED = 'credit-check-completed';
const SIGNAL_PROPERTY_VALUATION_COMPLETED = 'property-valuation-completed';
const QUERY_GET_APPLICATION = 'getApplication';

@Injectable()
export class MortgageService {
  protected readonly logger = new Logger(this.constructor.name);

  constructor(@Inject(WORKFLOW_CLIENT) private readonly client: Client) {}

  workflowId(applicationId: string): string {
    return `mortgage-application-${applicationId}`;
  }

  async completeCreditCheck(
    applicationId: string,
    result: CreditCheckResult,
    reference?: string,
  ): Promise<void> {
    if (!(await this.isWorkflowRunning(this.workflowId(applicationId)))) {
      throw new NotFoundException(`Application ${applicationId} not found`);
    }

    const handle = this.client.workflow.getHandle(
      this.workflowId(applicationId),
    );
    await handle.signal(SIGNAL_CREDIT_CHECK_COMPLETED, {
      applicationId,
      result,
      ...(reference !== undefined && { reference }),
    });
  }

  async completePropertyValuation(
    applicationId: string,
    valuationAmount: number,
    valuationReference: string,
  ): Promise<void> {
    if (!(await this.isWorkflowRunning(this.workflowId(applicationId)))) {
      throw new NotFoundException(`Application ${applicationId} not found`);
    }

    const handle = this.client.workflow.getHandle(
      this.workflowId(applicationId),
    );
    await handle.signal(SIGNAL_PROPERTY_VALUATION_COMPLETED, {
      applicationId,
      valuationAmount,
      valuationReference,
    });
  }

  async getApplication(applicationId: string): Promise<MortgageApplication> {
    const handle = this.client.workflow.getHandle(
      this.workflowId(applicationId),
    );
    try {
      return await handle.query<MortgageApplication>(QUERY_GET_APPLICATION);
    } catch (err) {
      if (err instanceof WorkflowNotFoundError) {
        throw new NotFoundException(`Application ${applicationId} not found`);
      }
      throw err;
    }
  }

  async isWorkflowRunning(workflowId: string): Promise<boolean> {
    this.logger.debug({ workflowId }, 'Checking if workflow is running');

    try {
      const desc = await this.client.workflow.getHandle(workflowId).describe();
      return (
        desc.status.code ===
        proto.temporal.api.enums.v1.WorkflowExecutionStatus
          .WORKFLOW_EXECUTION_STATUS_RUNNING
      );
    } catch {
      return false;
    }
  }

  async workflowExists(workflowId: string): Promise<boolean> {
    this.logger.debug({ workflowId }, 'Checking if workflow exists');

    try {
      await this.client.workflow.getHandle(workflowId).describe();
      return true;
    } catch {
      return false;
    }
  }

  async startApplication(
    applicationId: string,
    applicantName: string,
    scenario?: MortgageScenario,
  ): Promise<{ workflowId: string; applicationId: string }> {
    const workflowId = this.workflowId(applicationId);

    if (await this.workflowExists(workflowId)) {
      throw new ConflictException(
        `Workflow already exists for applicationId: ${applicationId}`,
      );
    }

    this.logger.log(
      { workflowId, applicationId },
      'Starting mortgage application workflow',
    );

    await this.client.workflow.start(WORKFLOW_TYPE, {
      taskQueue: TASK_QUEUE,
      workflowId,
      args: [
        {
          applicationId,
          applicantName,
          submittedAt: new Date().toISOString(),
          scenario: scenario ?? 'happy_path',
        },
      ],
    });

    return { workflowId, applicationId };
  }
}
