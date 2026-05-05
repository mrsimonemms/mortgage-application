import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import { Type } from 'class-transformer';
import {
  IsIn,
  IsObject,
  IsOptional,
  IsString,
  ValidateIf,
  ValidateNested,
} from 'class-validator';

import type { CreditCheckResult } from '../events/credit-check.event';

export class CreditCheckResultPayloadDto {
  @ApiProperty({
    enum: ['approved', 'rejected'],
    description: 'Credit check outcome',
  })
  @IsIn(['approved', 'rejected'])
  result: CreditCheckResult;

  @ApiPropertyOptional({
    description: 'Reference number from the credit bureau',
  })
  @IsOptional()
  @IsString()
  reference?: string;
}

export class ApplicationActionDto {
  @ApiProperty({
    enum: [
      'submit_credit_check_result',
      'retry_credit_check',
      'rerun_application',
    ],
    description: 'Action type',
  })
  @IsIn([
    'submit_credit_check_result',
    'retry_credit_check',
    'rerun_application',
  ])
  type:
    | 'submit_credit_check_result'
    | 'retry_credit_check'
    | 'rerun_application';

  @ApiPropertyOptional({
    description: 'Payload for submit_credit_check_result action',
    type: CreditCheckResultPayloadDto,
  })
  @ValidateIf(
    (o: ApplicationActionDto) => o.type === 'submit_credit_check_result',
  )
  @IsObject()
  @ValidateNested()
  @Type(() => CreditCheckResultPayloadDto)
  payload?: CreditCheckResultPayloadDto;
}
