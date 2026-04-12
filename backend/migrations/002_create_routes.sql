CREATE TABLE IF NOT EXISTS routes (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(30) UNIQUE,            -- NULL allowed for routes without a unique code
    type        VARCHAR(15) NOT NULL CHECK (type IN ('express', 'intercity', 'rail')),
    origin_id   INT NOT NULL REFERENCES terminals(id),
    dest_id     INT NOT NULL REFERENCES terminals(id),
    operator    VARCHAR(50),
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_routes_origin    ON routes(origin_id);
CREATE INDEX IF NOT EXISTS idx_routes_dest      ON routes(dest_id);
CREATE INDEX IF NOT EXISTS idx_routes_orig_dest ON routes(origin_id, dest_id);
