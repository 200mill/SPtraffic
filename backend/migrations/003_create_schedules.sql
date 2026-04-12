CREATE TABLE IF NOT EXISTS schedules (
    id              SERIAL PRIMARY KEY,
    route_id        INT NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    -- stored as minutes since midnight (0–1439) for timezone-free comparison
    departure_mins  SMALLINT NOT NULL,
    arrival_mins    SMALLINT,
    duration_min    SMALLINT,
    -- bitmask: bit0=Mon, bit1=Tue, ... bit6=Sun. 127 = everyday.
    days_of_week    SMALLINT NOT NULL DEFAULT 127,
    valid_from      DATE,
    valid_until     DATE,
    created_at      TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schedules_route ON schedules(route_id);
CREATE INDEX IF NOT EXISTS idx_schedules_dep   ON schedules(departure_mins);
