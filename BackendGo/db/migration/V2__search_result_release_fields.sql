-- V2__search_result_release_fields.sql
-- Persist the release metadata returned by Jackett so API search results do not drop it.

ALTER TABLE search_results
    ADD COLUMN IF NOT EXISTS guid VARCHAR(500),
    ADD COLUMN IF NOT EXISTS link TEXT,
    ADD COLUMN IF NOT EXISTS permalink TEXT,
    ADD COLUMN IF NOT EXISTS size BIGINT,
    ADD COLUMN IF NOT EXISTS pub_date TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS seeders INTEGER,
    ADD COLUMN IF NOT EXISTS leechers INTEGER,
    ADD COLUMN IF NOT EXISTS indexer VARCHAR(100);