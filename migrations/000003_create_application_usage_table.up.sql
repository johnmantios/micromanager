CREATE TABLE IF NOT EXISTS micromanager.application_usage(
    id serial PRIMARY KEY,
    user_id TEXT NOT NULL,
    date DATE NOT NULL,
    bundle_id TEXT NOT NULL,
    minutes_on NUMERIC(10, 1) NOT NULL
);
