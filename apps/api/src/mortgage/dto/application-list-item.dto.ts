import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import type { WorkflowExecutionStatusName } from '@temporalio/client/lib/types';

const WORKFLOW_EXECUTION_STATUS_VALUES: WorkflowExecutionStatusName[] = [
  'UNSPECIFIED',
  'RUNNING',
  'COMPLETED',
  'FAILED',
  'CANCELLED',
  'TERMINATED',
  'CONTINUED_AS_NEW',
  'TIMED_OUT',
  'PAUSED',
  'UNKNOWN',
];

export class ApplicationListItemDto {
  @ApiProperty({ description: 'Unique identifier for the application' })
  applicationId: string;

  @ApiProperty({ description: 'Full name of the applicant' })
  applicantName: string;

  @ApiPropertyOptional({ description: 'Demo scenario for this application' })
  scenario?: string;

  @ApiProperty({
    description: 'Temporal workflow execution status',
    enum: WORKFLOW_EXECUTION_STATUS_VALUES,
    example: 'RUNNING',
  })
  workflowStatus: WorkflowExecutionStatusName;
}
