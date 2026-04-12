CREATE TABLE IF NOT EXISTS terminals (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(20) UNIQUE NOT NULL,
    name        VARCHAR(100) NOT NULL,
    type        VARCHAR(10) NOT NULL CHECK (type IN ('bus', 'rail')),
    region_code VARCHAR(10) NOT NULL,
    lat         DOUBLE PRECISION NOT NULL,
    lon         DOUBLE PRECISION NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_terminals_region ON terminals(region_code);
CREATE INDEX IF NOT EXISTS idx_terminals_type   ON terminals(type);
