CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS machines (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL UNIQUE,
    os          text NOT NULL,
    token_hash  text NOT NULL,
    last_seen   timestamptz,
    kill_switch boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS jobs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    machine_id  uuid NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    type        text NOT NULL,
    payload     jsonb NOT NULL DEFAULT '{}'::jsonb,
    status      text NOT NULL DEFAULT 'queued',
    result      jsonb,
    priority    int  NOT NULL DEFAULT 0,
    ttl_at      timestamptz,
    created_by  text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    claimed_at  timestamptz,
    finished_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_jobs_claimable
    ON jobs (machine_id, status, priority DESC, created_at);

CREATE TABLE IF NOT EXISTS audit_log (
    id         bigserial PRIMARY KEY,
    job_id     uuid,
    machine_id uuid,
    event      text NOT NULL,
    detail     jsonb,
    at         timestamptz NOT NULL DEFAULT now()
);
