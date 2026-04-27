import { ApiProperty } from '@nestjs/swagger';
import { IsString } from 'class-validator';

import {
  MORTGAGE_EXAMPLE_APPLICANT_NAME,
  MORTGAGE_EXAMPLE_APPLICATION_ID,
} from '../constants';

export class StartMortgageApplicationDto {
  @ApiProperty({
    description: 'Unique identifier for the application',
    example: MORTGAGE_EXAMPLE_APPLICATION_ID,
  })
  @IsString()
  applicationId: string;

  @ApiProperty({
    description: 'Full name of the applicant',
    example: MORTGAGE_EXAMPLE_APPLICANT_NAME,
  })
  @IsString()
  applicantName: string;
}
