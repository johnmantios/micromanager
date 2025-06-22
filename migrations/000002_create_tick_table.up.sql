CREATE TABLE IF NOT EXISTS micromanager.tick(
    id serial PRIMARY KEY,
    user_id TEXT NOT NULL,
    is_locked BOOLEAN NOT NULL,
    tick TIMESTAMP WITH TIME ZONE
);