ALTER TABLE micromanager.application_usage
    DROP CONSTRAINT IF EXISTS application_usage_user_id_date_bundle_id_key;

ALTER TABLE micromanager.screentime
    DROP CONSTRAINT IF EXISTS screentime_user_id_date_key;

ALTER TABLE micromanager.application_usage DROP COLUMN IF EXISTS username;
