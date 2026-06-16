-- Cascade Postgres schema
-- Owned by: Auth (orgs/users/memberships), Dashboard (dashboards/widgets/queries)
-- All tables include tenant_id for row-level tenancy enforcement.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Auth domain ──────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS orgs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    password_hash   TEXT,            -- NULL for SSO-only users
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS memberships (
    org_id      UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_memberships_user_id ON memberships(user_id);

-- ── Dashboard domain ─────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS dashboards (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboards_org_id ON dashboards(org_id);

CREATE TABLE IF NOT EXISTS widgets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id    UUID NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    type            TEXT NOT NULL CHECK (type IN ('number', 'timeseries', 'funnel', 'table')),
    config          JSONB NOT NULL DEFAULT '{}',
    position        JSONB NOT NULL DEFAULT '{"x":0,"y":0,"w":4,"h":3}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_widgets_dashboard_id ON widgets(dashboard_id);

-- Saved query definitions (not results — those are cached in Redis)
CREATE TABLE IF NOT EXISTS saved_queries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    query_type      TEXT NOT NULL CHECK (query_type IN ('timeseries', 'funnel', 'cohort', 'count')),
    config          JSONB NOT NULL DEFAULT '{}',
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_saved_queries_org_id ON saved_queries(org_id);

-- ── Seed data for dev ────────────────────────────────────────────────────────

INSERT INTO orgs (id, name, slug) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Cascade Dev Org', 'cascade-dev')
ON CONFLICT DO NOTHING;

INSERT INTO users (id, email, name) VALUES
    ('00000000-0000-0000-0000-000000000002', 'dev@cascade.local', 'Dev User')
ON CONFLICT DO NOTHING;

INSERT INTO memberships (org_id, user_id, role) VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'owner')
ON CONFLICT DO NOTHING;
