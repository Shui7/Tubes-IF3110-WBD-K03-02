CREATE TABLE IF NOT EXISTS db_platform_migrations (
    id BIGSERIAL PRIMARY KEY,
    version BIGINT NOT NULL,
    sql_text TEXT NOT NULL,
    executed_sql TEXT NOT NULL DEFAULT '',
    sql_down TEXT NOT NULL DEFAULT '',
    is_irreversible BOOLEAN NOT NULL DEFAULT FALSE,
    irreversible_reason TEXT,
    action TEXT NOT NULL DEFAULT 'up',
    target_execution_id BIGINT,
    executed_by TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('success', 'failed')),
    error_message TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS sql_down TEXT NOT NULL DEFAULT '';
ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS executed_sql TEXT NOT NULL DEFAULT '';
ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS is_irreversible BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS irreversible_reason TEXT;
ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS action TEXT NOT NULL DEFAULT 'up';
ALTER TABLE db_platform_migrations ADD COLUMN IF NOT EXISTS target_execution_id BIGINT;
UPDATE db_platform_migrations SET executed_sql = sql_text WHERE executed_sql = '';
UPDATE db_platform_migrations
SET irreversible_reason = 'Legacy migration predates stored SQL down definitions.'
WHERE is_irreversible = TRUE AND (irreversible_reason IS NULL OR irreversible_reason = '');
