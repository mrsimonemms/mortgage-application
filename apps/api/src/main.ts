import {
  ClassSerializerInterceptor,
  ConsoleLogger,
  Logger,
  ValidationPipe,
  VersioningType,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { NestFactory, Reflector } from '@nestjs/core';
import {
  FastifyAdapter,
  NestFastifyApplication,
} from '@nestjs/platform-fastify';

import { AppModule } from './app.module';
import loggerConfig from './config/logger';

const logger = new Logger('bootstrap');

async function bootstrap() {
  const app = await NestFactory.create<NestFastifyApplication>(
    AppModule,
    new FastifyAdapter(),
    { logger: new ConsoleLogger(loggerConfig()) },
  );

  const config = app.get(ConfigService);

  app
    .enableShutdownHooks()
    .enableVersioning({
      type: VersioningType.URI,
      defaultVersion: '1',
    })
    .useGlobalPipes(
      new ValidationPipe({
        transform: true,
        whitelist: true,
      }),
    )
    .useGlobalInterceptors(new ClassSerializerInterceptor(app.get(Reflector)));

  await app.listen(
    config.getOrThrow<number>('server.port'),
    config.getOrThrow('server.host'),
  );
}

bootstrap().catch((err: Error) => {
  /* Unlikely to get to here but a final catchall */
  logger.fatal(err.stack);
});
