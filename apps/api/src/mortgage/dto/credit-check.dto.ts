import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import { IsIn, IsOptional, IsString } from 'class-validator';

import type { CreditCheckResult } from '../events/credit-check.event';

export class CreditCheckDto {
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
