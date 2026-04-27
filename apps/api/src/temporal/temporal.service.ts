/*
 * Copyright 2025 Simon Emms <simon@simonemms.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
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
