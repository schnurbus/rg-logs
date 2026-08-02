-- GearScore (GearScoreLite / WotLK) from Rising Gods profiler at ingest time.
ALTER TABLE actors ADD COLUMN IF NOT EXISTS gear_score INT;
