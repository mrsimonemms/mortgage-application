import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import { IsIn, IsInt, IsOptional, IsString, Max, Min } from 'class-validator';

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
    enum: MORTGAGE_SCENARIOS.map((s) => s.name),
    example: 'happy_path',
  })
  @IsOptional()
  @IsIn(MORTGAGE_SCENARIOS.map((s) => s.name))
  scenario?: MortgageScenario;

  @ApiPropertyOptional({
    description:
      'Temporal demo control: probability (0–75) that each eligible activity fails on a given attempt. Temporal retries absorb transient failures. Defaults to 0 (no injected failures).',
    minimum: 0,
    maximum: 75,
    example: 0,
  })
  @IsOptional()
  @IsInt()
  @Min(0)
  @Max(75)
  externalFailureRatePercent?: number;
}
