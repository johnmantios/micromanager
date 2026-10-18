-- Carry the username alongside application usage, matching screentime.
ALTER TABLE micromanager.application_usage ADD COLUMN IF NOT EXISTS username TEXT;

-- Enforce one row per identity/day (screentime) and one per identity/day/app
-- (application_usage) so the poller can upsert instead of inserting duplicates.
ALTER TABLE micromanager.screentime
    ADD CONSTRAINT screentime_user_id_date_key UNIQUE (user_id, date);

ALTER TABLE micromanager.application_usage
    ADD CONSTRAINT application_usage_user_id_date_bundle_id_key UNIQUE (user_id, date, bundle_id);
