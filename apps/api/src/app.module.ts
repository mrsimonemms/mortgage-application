import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';

import config from './config';
import { HealthModule } from './health/health.module';
import { TemporalModule } from './temporal/temporal.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      load: config,
    }),

    HealthModule,
    TemporalModule,
  ],
})
export class AppModule {}
