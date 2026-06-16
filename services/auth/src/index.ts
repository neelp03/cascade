import Fastify from 'fastify';
import fastifyJwt from '@fastify/jwt';
import { authRoutes } from './routes/auth.js';

const jwtSecret = process.env.JWT_SECRET;
if (!jwtSecret || jwtSecret.length < 32) {
  console.error('FATAL: JWT_SECRET must be set and at least 32 characters');
  process.exit(1);
}

const app = Fastify({ logger: true });

await app.register(fastifyJwt, { secret: jwtSecret });

app.get('/health', async () => ({ status: 'ok' }));

await app.register(authRoutes);

const port = parseInt(process.env.PORT ?? '8083', 10);
try {
  await app.listen({ port, host: '0.0.0.0' });
} catch (err) {
  app.log.error(err);
  process.exit(1);
}
