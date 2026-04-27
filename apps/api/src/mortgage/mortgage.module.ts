import { Module } from '@nestjs/common';

import { TemporalModule } from '../temporal/temporal.module';
import { MortgageController } from './mortgage.controller';
import { MortgageService } from './mortgage.service';

@Module({
  imports: [TemporalModule],
  controllers: [MortgageController],
  providers: [MortgageService],
})
export class MortgageModule {}
