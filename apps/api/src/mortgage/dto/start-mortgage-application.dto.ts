import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import { IsIn, IsOptional, IsString } from 'class-validator';

import {
  MORTGAGE_EXAMPLE_APPLICANT_NAME,
  MORTGAGE_EXAMPLE_APPLICATION_ID,
} from '../constants';
import { MORTGAGE_SCENARIOS } from '../models/mortgage-scenario.type';
import type { MortgageScenario } from '../models/mortgage-scenario.type';

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

  @ApiPropertyOptional({
    description:
      'Demo scenario to run. Defaults to happy_path when omitted. Use fail_after_offer_reservation to trigger a deliberate completion failure after offer reservation.',
    enum: MORTGAGE_SCENARIOS,
    example: 'happy_path',
  })
  @IsOptional()
  @IsIn(MORTGAGE_SCENARIOS)
  scenario?: MortgageScenario;
}
