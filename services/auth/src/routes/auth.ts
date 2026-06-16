import type { FastifyInstance } from 'fastify';
import bcrypt from 'bcryptjs';
import { z } from 'zod';
import {
  findUserByEmail,
  createOrgUserMembership,
  findOrgByMembership,
} from '../db/queries.js';

const RegisterBody = z.object({
  email: z.string().email(),
  password: z.string().min(8),
  name: z.string().min(1),
  org_name: z.string().min(1),
});

const LoginBody = z.object({
  email: z.string().email(),
  password: z.string(),
});

function slugify(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
}

export async function authRoutes(app: FastifyInstance) {
  app.post('/auth/register', async (req, reply) => {
    const parsed = RegisterBody.safeParse(req.body);
    if (!parsed.success) {
      return reply.status(400).send({ error: parsed.error.issues[0].message });
    }
    const { email, password, name, org_name } = parsed.data;

    const existing = await findUserByEmail(email);
    if (existing) {
      return reply.status(409).send({ error: 'email already registered' });
    }

    const passwordHash = await bcrypt.hash(password, 12);
    const slug = slugify(org_name);

    let org, user;
    try {
      ({ org, user } = await createOrgUserMembership(org_name, slug, email, name, passwordHash));
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      if (msg.includes('unique') || msg.includes('duplicate')) {
        return reply.status(409).send({ error: 'org slug already taken' });
      }
      throw err;
    }

    const token = await reply.jwtSign(
      { sub: user.id, org_id: org.id, role: 'owner' },
      { expiresIn: '7d' },
    );

    return reply.status(201).send({
      token,
      user: { id: user.id, email: user.email, name: user.name },
      org: { id: org.id, name: org.name, slug: org.slug },
    });
  });

  app.post('/auth/login', async (req, reply) => {
    const parsed = LoginBody.safeParse(req.body);
    if (!parsed.success) {
      return reply.status(400).send({ error: parsed.error.issues[0].message });
    }
    const { email, password } = parsed.data;

    const user = await findUserByEmail(email);
    if (!user || !user.password_hash) {
      return reply.status(401).send({ error: 'invalid credentials' });
    }

    const valid = await bcrypt.compare(password, user.password_hash);
    if (!valid) {
      return reply.status(401).send({ error: 'invalid credentials' });
    }

    const org = await findOrgByMembership(user.id);
    if (!org) {
      return reply.status(403).send({ error: 'no org membership' });
    }

    const token = await reply.jwtSign(
      { sub: user.id, org_id: org.id, role: 'owner' },
      { expiresIn: '7d' },
    );

    return reply.send({
      token,
      user: { id: user.id, email: user.email, name: user.name },
      org: { id: org.id, name: org.name, slug: org.slug },
    });
  });
}
