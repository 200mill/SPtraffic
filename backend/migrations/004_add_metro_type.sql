-- Widen CHECK constraints to accept the new 'metro' transport type (도시철도/지하철).
ALTER TABLE terminals DROP CONSTRAINT IF EXISTS terminals_type_check;
ALTER TABLE terminals ADD CONSTRAINT terminals_type_check
  CHECK (type IN ('bus', 'rail', 'metro'));

ALTER TABLE routes DROP CONSTRAINT IF EXISTS routes_type_check;
ALTER TABLE routes ADD CONSTRAINT routes_type_check
  CHECK (type IN ('express', 'intercity', 'rail', 'metro'));
