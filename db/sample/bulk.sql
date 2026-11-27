SET client_min_messages = warning;
SET synchronous_commit = off;

DROP TABLE IF EXISTS bulk_orders;

CREATE TABLE bulk_orders (
    id          BIGINT        PRIMARY KEY,
    customer_id INTEGER       NOT NULL,
    status      TEXT          NOT NULL,
    amount      NUMERIC(12,2) NOT NULL,
    score       INTEGER       NOT NULL DEFAULT 0,
    note        TEXT          NOT NULL,
    created_at  TIMESTAMPTZ   NOT NULL
);

INSERT INTO bulk_orders (id, customer_id, status, amount, note, created_at)
SELECT g,
       (g % 100000)::int + 1,
       (ARRAY['pending', 'paid', 'shipped', 'cancelled'])[(g % 4)::int + 1],
       (((g * 7919) % 100000)::numeric / 100)::numeric(12,2),
       md5(g::text) || md5((g * 7)::text),
       TIMESTAMPTZ '2026-01-01 00:00:00+00' + ((g % 31536000)::int * INTERVAL '1 second')
FROM generate_series(1::bigint, 3000000::bigint) AS g;

-- Secondary indexes make a full-table UPDATE slower (every new row version is indexed again)
CREATE INDEX bulk_orders_customer_idx ON bulk_orders (customer_id);
CREATE INDEX bulk_orders_status_idx   ON bulk_orders (status);

ANALYZE bulk_orders;
