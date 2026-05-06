import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';

import { APPLICATION_WORKFLOW_STATUSES } from '../models/application-workflow-status.type';
import type { ApplicationWorkflowStatus } from '../models/application-workflow-status.type';

export class ApplicationListItemDto {
  @ApiProperty({ description: 'Unique identifier for the application' })
  applicationId: string;

  @ApiProperty({ description: 'Full name of the applicant' })
  applicantName: string;

  @ApiPropertyOptional({ description: 'Demo scenario for this application' })
  scenario?: string;

  @ApiProperty({
    description:
      'Application-level workflow lifecycle status, normalised from the underlying Temporal execution status',
    enum: APPLICATION_WORKFLOW_STATUSES,
    example: 'running',
  })
  workflowStatus: ApplicationWorkflowStatus;
}
