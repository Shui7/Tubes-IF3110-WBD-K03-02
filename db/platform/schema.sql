CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    password VARCHAR(255),
    name VARCHAR(100) NOT NULL DEFAULT '',
    role VARCHAR(20) NOT NULL DEFAULT 'User',
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(128) PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    csrf_token VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    previous_id VARCHAR(128),
    rotated_at TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    leader_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_groups (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, group_id)
);

CREATE TABLE IF NOT EXISTS database_connections (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    database_name VARCHAR(100) NOT NULL,
    dbname VARCHAR(100),
    host VARCHAR(255) NOT NULL,
    port INT NOT NULL DEFAULT 5432,
    username VARCHAR(100) NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_database_access (
    id SERIAL PRIMARY KEY,
    group_id INT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    database_id INT REFERENCES database_connections(id) ON DELETE CASCADE,
    db_id INT REFERENCES database_connections(id) ON DELETE CASCADE,
    access_level VARCHAR(20) NOT NULL DEFAULT 'view',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS migration_change_requests (
    id BIGSERIAL PRIMARY KEY,
    database_id INT NOT NULL REFERENCES database_connections(id),
    requester_user_id INT NOT NULL,
    requester_name TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('up', 'rollback', 'forward')),
    sql_up TEXT NOT NULL DEFAULT '',
    sql_down TEXT NOT NULL DEFAULT '',
    is_irreversible BOOLEAN NOT NULL DEFAULT FALSE,
    irreversible_reason TEXT,
    target_execution_id BIGINT,
    mode TEXT NOT NULL DEFAULT 'plain' CHECK (mode IN ('plain', 'batch')),
    batch_size INT,
    batch_table TEXT,
    batch_column TEXT,
    status TEXT NOT NULL CHECK (status IN ('draft', 'pending_review', 'approved', 'pending_queue', 'running', 'success', 'failed', 'rejected', 'partial')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS migration_change_approvals (
    id BIGSERIAL PRIMARY KEY,
    request_id BIGINT NOT NULL REFERENCES migration_change_requests(id) ON DELETE CASCADE,
    group_id INT NOT NULL,
    group_name TEXT NOT NULL,
    leader_user_id INT,
    leader_name TEXT NOT NULL,
    decision TEXT NOT NULL DEFAULT 'pending' CHECK (decision IN ('pending', 'approved', 'rejected')),
    comment TEXT NOT NULL DEFAULT '',
    responded_at TIMESTAMPTZ,
    UNIQUE (request_id, group_id)
);

CREATE TABLE IF NOT EXISTS migration_change_events (
    id BIGSERIAL PRIMARY KEY,
    request_id BIGINT NOT NULL REFERENCES migration_change_requests(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    actor_user_id INT,
    actor_name TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
