import { AESCodec } from '@mrsimonemms/temporal-codec-server';
import { Provider } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import {
  Client,
  ClientOptions,
  Connection,
  ConnectionOptions,
} from '@temporalio/client';

export const CONNECTION = Symbol('CONNECTION');
export const WORKFLOW_CLIENT = Symbol('WORKFLOW_CLIENT');

export const temporalProviders: Provider[] = [
  {
    inject: [ConfigService],
    provide: CONNECTION,
    useFactory(cfg: ConfigService): Promise<Connection> {
      return Connection.connect(
        cfg.getOrThrow<ConnectionOptions>('temporal.connection'),
      );
    },
  },
  {
    inject: [CONNECTION, ConfigService],
    provide: WORKFLOW_CLIENT,
    useFactory: async (
      connection: Connection,
      cfg: ConfigService,
    ): Promise<Client> => {
      const opts: ClientOptions = { connection };

      const encryptionKeyPath = cfg.get<string>('temporal.encryptionKeyPath');
      if (encryptionKeyPath) {
        opts.dataConverter = {
          payloadCodecs: [await AESCodec.create(encryptionKeyPath)],
        };
      }

      return new Client(opts);
    },
  },
];
