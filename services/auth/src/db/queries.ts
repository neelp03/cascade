import { pool } from './pool.js';

export interface OrgRow {
  id: string;
  name: string;
  slug: string;
}

export interface UserRow {
  id: string;
  email: string;
  name: string;
  password_hash: string | null;
}

export async function findUserByEmail(email: string): Promise<UserRow | null> {
  const res = await pool.query<UserRow>(
    'SELECT id, email, name, password_hash FROM users WHERE email = $1',
    [email],
  );
  return res.rows[0] ?? null;
}

export async function createOrgUserMembership(
  orgName: string,
  orgSlug: string,
  email: string,
  name: string,
  passwordHash: string,
): Promise<{ org: OrgRow; user: UserRow }> {
  const client = await pool.connect();
  try {
    await client.query('BEGIN');

    const orgRes = await client.query<OrgRow>(
      'INSERT INTO orgs (name, slug) VALUES ($1, $2) RETURNING id, name, slug',
      [orgName, orgSlug],
    );
    const org = orgRes.rows[0];

    const userRes = await client.query<UserRow>(
      'INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING id, email, name, password_hash',
      [email, name, passwordHash],
    );
    const user = userRes.rows[0];

    await client.query(
      'INSERT INTO memberships (org_id, user_id, role) VALUES ($1, $2, $3)',
      [org.id, user.id, 'owner'],
    );

    await client.query('COMMIT');
    return { org, user };
  } catch (err) {
    await client.query('ROLLBACK');
    throw err;
  } finally {
    client.release();
  }
}

export async function findOrgByMembership(userId: string): Promise<OrgRow | null> {
  const res = await pool.query<OrgRow>(
    `SELECT o.id, o.name, o.slug
     FROM orgs o
     JOIN memberships m ON m.org_id = o.id
     WHERE m.user_id = $1
     ORDER BY m.created_at ASC
     LIMIT 1`,
    [userId],
  );
  return res.rows[0] ?? null;
}
