import { Controller, Get, Inject, VERSION_NEUTRAL } from '@nestjs/common';
import { HealthCheck, HealthCheckService } from '@nestjs/terminus';

import { TemporalService } from '../temporal/temporal.service';

@Controller({
  path: 'health',
  version: VERSION_NEUTRAL,
})
export class HealthController {
  @Inject(HealthCheckService)
  private readonly health: HealthCheckService;

  @Inject(TemporalService)
  private readonly temporal: TemporalService;

  @Get()
  @HealthCheck()
  check() {
    // Allow 1 second before timeout
    const timeout = 1000;

    return this.health.check([() => this.temporal.healthcheck(timeout)]);
  }
}
