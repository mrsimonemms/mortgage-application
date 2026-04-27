import {
  Inject,
  Injectable,
  Logger,
  OnApplicationShutdown,
} from '@nestjs/common';
import { HealthIndicatorResult, HealthIndicatorStatus } from '@nestjs/terminus';
import { Connection } from '@temporalio/client';
import { grpc as grpcProto } from '@temporalio/proto';
import { setTimeout } from 'timers/promises';

import { CONNECTION } from './temporal.providers';

@Injectable()
export class TemporalService implements OnApplicationShutdown {
  protected readonly logger = new Logger(this.constructor.name);

  @Inject(CONNECTION)
  private connection: Connection;

  // Close connection once there are no further connections possible
  async onApplicationShutdown() {
    this.logger.log('Disconnecting from Temporal server');
    await this.connection.close();
  }

  async healthcheck(
    timeout: number = 1000,
    serviceName = 'temporal',
  ): Promise<HealthIndicatorResult> {
    let status: HealthIndicatorStatus = 'down';

    try {
      const healthcheck =
        await Promise.race<grpcProto.health.v1.HealthCheckResponse>([
          this.connection.healthService.check({}),
          setTimeout(timeout).then(() => {
            throw new Error(`${serviceName} timeout`);
          }),
        ]);

      if (
        healthcheck.status ===
        grpcProto.health.v1.HealthCheckResponse.ServingStatus.SERVING
      ) {
        status = 'up';
      }
    } catch (err) {
      this.logger.error('Temporal unhealthy', err);
    }

    return {
      [serviceName]: {
        status,
      },
    };
  }
}
