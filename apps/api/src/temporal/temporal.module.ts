import { Module } from '@nestjs/common';

import { temporalProviders } from './temporal.providers';
import { TemporalService } from './temporal.service';

@Module({
  providers: [...temporalProviders, TemporalService],
  exports: [...temporalProviders, TemporalService],
})
export class TemporalModule {}
