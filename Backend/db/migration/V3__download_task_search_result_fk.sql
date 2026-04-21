-- V3__download_task_search_result_fk.sql
-- Add the missing FK for download_tasks.search_result_id.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_download_tasks_search_result'
    ) THEN
        ALTER TABLE download_tasks
            ADD CONSTRAINT fk_download_tasks_search_result
            FOREIGN KEY (search_result_id)
            REFERENCES search_results(id)
            ON DELETE CASCADE;
    END IF;
END $$;