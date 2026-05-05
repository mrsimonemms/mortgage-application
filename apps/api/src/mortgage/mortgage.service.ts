import {
  BadRequestException,
  ConflictException,
  Inject,
  Injectable,
  Logger,
  NotFoundException,
} from '@nestjs/common';
import { Client, WorkflowNotFoundError } from '@temporalio/client';
import type {
  WorkflowExecutionInfo,
  WorkflowExecutionStatusName,
} from '@temporalio/client/lib/types';
import * as proto from '@temporalio/proto';
import { randomUUID } from 'node:crypto';

import { WORKFLOW_CLIENT } from '../temporal/temporal.providers';
import { ApplicationActionDto } from './dto/application-action.dto';
import { ApplicationListItemDto } from './dto/application-list-item.dto';
import { CreditCheckResult } from './events/credit-check.event';
import { MortgageApplication } from './models/mortgage-application.model';
import { MortgageScenario } from './models/mortgage-scenario.type';

// The workflow type name is the short function name that the Temporal Go SDK
// derives from runtime.FuncForPC. It must match the Go worker registration.
const WORKFLOW_TYPE = 'MortgageApplicationWorkflow';
const TASK_QUEUE = 'mortgage-application';
const SIGNAL_CREDIT_CHECK_COMPLETED = 'credit-check-completed';
const SIGNAL_RETRY_CREDIT_CHECK = 'retry-credit-check';
const QUERY_GET_APPLICATION = 'getApplication';

const STATUS_RUNNING =
  proto.temporal.api.enums.v1.WorkflowExecutionStatus
    .WORKFLOW_EXECUTION_STATUS_RUNNING;

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
      return desc.status.code === STATUS_RUNNING;
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
    externalFailureRatePercent?: number,
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

    const resolvedScenario = scenario ?? 'happy_path';
    const resolvedFailureRate = this.allowsFailureInjection(resolvedScenario)
      ? (externalFailureRatePercent ?? 0)
      : 0;

    await this.client.workflow.start(WORKFLOW_TYPE, {
      taskQueue: TASK_QUEUE,
      workflowId,
      memo: {
        applicationId,
        applicantName,
        scenario: resolvedScenario,
        externalFailureRatePercent: resolvedFailureRate,
      },
      args: [
        {
          applicationId,
          applicantName,
          submittedAt: new Date().toISOString(),
          scenario: resolvedScenario,
          externalFailureRatePercent: resolvedFailureRate,
        },
      ],
    });

    return { workflowId, applicationId };
  }

  async retryCreditCheck(applicationId: string): Promise<void> {
    if (!(await this.isWorkflowRunning(this.workflowId(applicationId)))) {
      throw new NotFoundException(`Application ${applicationId} not found`);
    }

    this.logger.log(
      { applicationId },
      'Operator retry: sending retry-credit-check signal',
    );

    const handle = this.client.workflow.getHandle(
      this.workflowId(applicationId),
    );
    await handle.signal(SIGNAL_RETRY_CREDIT_CHECK);
  }

  async rerunApplication(
    applicationId: string,
  ): Promise<{ applicationId: string; workflowId: string }> {
    const existingWorkflowId = this.workflowId(applicationId);

    let applicantName = '';
    let scenario = 'happy_path';
    let externalFailureRatePercent = 0;

    try {
      const desc = await this.client.workflow
        .getHandle(existingWorkflowId)
        .describe();
      const memo = this.readMemo(desc.memo);
      applicantName = memo.applicantName ?? '';
      scenario = memo.scenario ?? 'happy_path';
      externalFailureRatePercent = this.allowsFailureInjection(scenario)
        ? (memo.externalFailureRatePercent ?? 0)
        : 0;
    } catch (err) {
      if (err instanceof WorkflowNotFoundError) {
        throw new NotFoundException(`Application ${applicationId} not found`);
      }
      throw err;
    }

    const newApplicationId = randomUUID();
    const newWorkflowId = this.workflowId(newApplicationId);

    this.logger.log(
      { applicationId, newApplicationId },
      'Operator rerun: starting new workflow',
    );

    await this.client.workflow.start(WORKFLOW_TYPE, {
      taskQueue: TASK_QUEUE,
      workflowId: newWorkflowId,
      memo: {
        applicationId: newApplicationId,
        applicantName,
        scenario,
        externalFailureRatePercent,
      },
      args: [
        {
          applicationId: newApplicationId,
          applicantName,
          submittedAt: new Date().toISOString(),
          scenario,
          originalApplicationId: applicationId,
          externalFailureRatePercent,
        },
      ],
    });

    return { applicationId: newApplicationId, workflowId: newWorkflowId };
  }

  async handleAction(
    applicationId: string,
    action: ApplicationActionDto,
  ): Promise<{ applicationId: string; workflowId: string } | void> {
    switch (action.type) {
      case 'submit_credit_check_result':
        if (!action.payload) {
          throw new BadRequestException(
            'payload is required for submit_credit_check_result',
          );
        }
        return this.completeCreditCheck(
          applicationId,
          action.payload.result,
          action.payload.reference,
        );
      case 'retry_credit_check':
        return this.retryCreditCheck(applicationId);
      case 'rerun_application':
        return this.rerunApplication(applicationId);
    }
  }

  async listApplications(): Promise<ApplicationListItemDto[]> {
    const applications: ApplicationListItemDto[] = [];

    try {
      for await (const info of this.client.workflow.list({
        query: `WorkflowType = "${WORKFLOW_TYPE}"`,
      })) {
        try {
          applications.push(await this.resolveListItem(info));
        } catch {
          this.logger.warn(
            { workflowId: info.workflowId },
            'Failed to retrieve application details',
          );
        }
      }
    } catch {
      this.logger.warn('Failed to list applications from Temporal');
    }

    return applications;
  }

  private async resolveListItem(
    info: WorkflowExecutionInfo,
  ): Promise<ApplicationListItemDto> {
    const workflowStatus = info.status.name;
    const memo = this.readMemo(info.memo);

    if (memo.applicantName !== undefined) {
      return this.toApplicationListItem(info.workflowId, workflowStatus, memo);
    }

    // Legacy workflows without memo — fall back to query or result
    const handle = this.client.workflow.getHandle(info.workflowId);

    if (info.status.name === 'RUNNING') {
      const app = await handle.query<MortgageApplication>(
        QUERY_GET_APPLICATION,
      );
      return this.toApplicationListItem(info.workflowId, workflowStatus, {
        applicationId: app.applicationId,
        applicantName: app.applicantName,
      });
    }

    if (info.status.name === 'COMPLETED') {
      const app = (await handle.result()) as MortgageApplication;
      return this.toApplicationListItem(info.workflowId, workflowStatus, {
        applicationId: app.applicationId,
        applicantName: app.applicantName,
      });
    }

    return this.toApplicationListItem(info.workflowId, workflowStatus, {});
  }

  private allowsFailureInjection(scenario: string): boolean {
    return scenario === 'happy_path';
  }

  private readMemo(memo: Record<string, unknown> | undefined): {
    applicationId?: string;
    applicantName?: string;
    scenario?: string;
    externalFailureRatePercent?: number;
  } {
    if (!memo) return {};
    return {
      applicationId:
        typeof memo['applicationId'] === 'string'
          ? memo['applicationId']
          : undefined,
      applicantName:
        typeof memo['applicantName'] === 'string'
          ? memo['applicantName']
          : undefined,
      scenario:
        typeof memo['scenario'] === 'string' ? memo['scenario'] : undefined,
      externalFailureRatePercent:
        typeof memo['externalFailureRatePercent'] === 'number'
          ? memo['externalFailureRatePercent']
          : undefined,
    };
  }

  private toApplicationListItem(
    workflowId: string,
    workflowStatus: WorkflowExecutionStatusName,
    data: { applicationId?: string; applicantName?: string; scenario?: string },
  ): ApplicationListItemDto {
    return {
      applicationId:
        data.applicationId ?? workflowId.replace('mortgage-application-', ''),
      applicantName: data.applicantName ?? '',
      scenario: data.scenario,
      workflowStatus,
    };
  }
}
