import { registerAs } from '@nestjs/config';

export default registerAs('server', () => ({
  host: process.env.HOST ?? '0.0.0.0',
  port: Number(process.env.PORT ?? 3000),
}));
