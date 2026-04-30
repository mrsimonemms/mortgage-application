import { ApiProperty } from '@nestjs/swagger';
import { IsNumber, IsString } from 'class-validator';

export class PropertyValuationDto {
  @ApiProperty({
    description: 'Assessed property value in the base currency unit',
    example: 350000,
  })
  @IsNumber()
  valuationAmount: number;

  @ApiProperty({
    description: 'Reference number assigned by the external valuer',
    example: 'VAL-APP-001',
  })
  @IsString()
  valuationReference: string;
}
