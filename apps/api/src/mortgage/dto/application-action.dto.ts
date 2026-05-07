import { ApiProperty, ApiPropertyOptional } from '@nestjs/swagger';
import { Type } from 'class-transformer';
import {
  IsIn,
  IsNumber,
  IsObject,
  IsOptional,
  IsPositive,
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

// PropertyValuationPayloadDto carries the operator-supplied property value
// for the v2 workflow. The API enforces:
//   * propertyValue is required (no API-side default)
//   * must be a number (rejects strings/booleans)
//   * must be positive (a non-positive valuation is treated as a wiring bug)
// These constraints match the worker's invariants so the workflow only ever
// sees positive values via the property-valuation-submitted signal.
export class PropertyValuationPayloadDto {
  @ApiProperty({
    description: 'Property value in pounds (positive number)',
    example: 350000,
  })
  @IsNumber()
  @IsPositive()
  propertyValue: number;
}

const ACTION_TYPES = [
  'submit_credit_check_result',
  'retry_credit_check',
  'rerun_application',
  'submit_property_valuation',
] as const;

export type ApplicationActionType = (typeof ACTION_TYPES)[number];

export class ApplicationActionDto {
  @ApiProperty({
    enum: ACTION_TYPES,
    description: 'Action type',
  })
  @IsIn(ACTION_TYPES)
  type: ApplicationActionType;

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

  @ApiPropertyOptional({
    description: 'Payload for submit_property_valuation action',
    type: PropertyValuationPayloadDto,
  })
  @ValidateIf(
    (o: ApplicationActionDto) => o.type === 'submit_property_valuation',
  )
  @IsObject()
  @ValidateNested()
  @Type(() => PropertyValuationPayloadDto)
  propertyValuation?: PropertyValuationPayloadDto;
}
