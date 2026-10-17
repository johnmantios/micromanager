CREATE TABLE IF NOT EXISTS micromanager.screentime(
    id serial PRIMARY KEY,
    user_id TEXT NOT NULL,
    username TEXT ,
    date DATE NOT NULL,
    minutes_on INT NOT NULL
);