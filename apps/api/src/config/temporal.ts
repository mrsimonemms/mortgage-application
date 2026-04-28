import { registerAs } from '@nestjs/config';
import { ConnectionOptions } from '@temporalio/client';

export default registerAs(
  'temporal',
  (): { connection: ConnectionOptions; encryptionKeyPath?: string } => ({
    connection: {
      address: process.env.TEMPORAL_ADDRESS,
      apiKey: process.env.TEMPORAL_KEY,
      connectTimeout: process.env.TEMPORAL_CONNECTION_TIMEOUT,
      tls: process.env.TEMPORAL_TLS === 'true',
      metadata: {
        'temporal-namespace': process.env.TEMPORAL_NAMESPACE ?? 'default',
      },
    },
    encryptionKeyPath: process.env.KEYS_PATH,
  }),
);
