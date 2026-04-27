import { Provider } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Client, Connection, ConnectionOptions } from '@temporalio/client';

export const CONNECTION = Symbol('CONNECTION');
export const WORKFLOW_CLIENT = Symbol('WORKFLOW_CLIENT');

export const temporalProviders: Provider[] = [
  {
    inject: [ConfigService],
    provide: CONNECTION,
    useFactory(cfg: ConfigService): Promise<Connection> {
      return Connection.connect(cfg.getOrThrow<ConnectionOptions>('temporal'));
    },
  },
  {
    inject: [CONNECTION],
    provide: WORKFLOW_CLIENT,
    useFactory: (connection: Connection): Client => {
      return new Client({ connection });
    },
  },
];
