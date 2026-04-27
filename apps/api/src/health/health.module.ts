import { Module } from '@nestjs/common';
import { TerminusModule } from '@nestjs/terminus';

import { TemporalModule } from '../temporal/temporal.module';
import { HealthController } from './health.controller';

@Module({
  imports: [TerminusModule, TemporalModule],
  controllers: [HealthController],
})
export class HealthModule {}
