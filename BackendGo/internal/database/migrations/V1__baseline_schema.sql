-- V1__baseline_schema.sql
-- Initial database schema for Netflix Torrent Backend
-- This migration creates all tables needed for the application

-- Movies table (catalog)
CREATE TABLE IF NOT EXISTS movies (
    id BIGSERIAL PRIMARY KEY,
    tmdb_id INTEGER UNIQUE,
    title VARCHAR(255) NOT NULL,
    overview TEXT,
    poster_path VARCHAR(500),
    backdrop_path VARCHAR(500),
    release_date VARCHAR(10),
    vote_average DOUBLE PRECISION,
    vote_count INTEGER,
    popularity DOUBLE PRECISION,
    genre_ids TEXT,
    adult BOOLEAN DEFAULT FALSE,
    original_language VARCHAR(10),
    original_title VARCHAR(255),
    catalog_added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id);
CREATE INDEX IF NOT EXISTS idx_movies_title ON movies(title);
CREATE INDEX IF NOT EXISTS idx_movies_popularity ON movies(popularity DESC);

-- Storage profiles table
CREATE TABLE IF NOT EXISTS storage_profiles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    base_path VARCHAR(1000) NOT NULL,
    priority INTEGER NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Search jobs table
CREATE TABLE IF NOT EXISTS search_jobs (
    id BIGSERIAL PRIMARY KEY,
    query VARCHAR(500) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'REQUESTED',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_search_jobs_status ON search_jobs(status);
CREATE INDEX IF NOT EXISTS idx_search_jobs_created_at ON search_jobs(created_at);

-- Search results table
CREATE TABLE IF NOT EXISTS search_results (
    id BIGSERIAL PRIMARY KEY,
    search_job_id BIGINT NOT NULL REFERENCES search_jobs(id) ON DELETE CASCADE,
    tmdb_id INTEGER,
    title VARCHAR(500) NOT NULL,
    year INTEGER,
    score DOUBLE PRECISION,
    provider VARCHAR(100),
    hash VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_search_results_job_id ON search_results(search_job_id);

-- Download tasks table
CREATE TABLE IF NOT EXISTS download_tasks (
    id BIGSERIAL PRIMARY KEY,
    search_result_id BIGINT NOT NULL,
    torrent_hash VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    progress DOUBLE PRECISION NOT NULL DEFAULT 0,
    speed BIGINT,
    peer_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_download_tasks_status ON download_tasks(status);
CREATE INDEX IF NOT EXISTS idx_download_tasks_hash ON download_tasks(torrent_hash);

-- Download state transitions table (audit trail)
CREATE TABLE IF NOT EXISTS download_state_transitions (
    id BIGSERIAL PRIMARY KEY,
    download_task_id BIGINT NOT NULL REFERENCES download_tasks(id) ON DELETE CASCADE,
    from_status VARCHAR(50),
    to_status VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reason VARCHAR(500)
);

CREATE INDEX IF NOT EXISTS idx_download_transitions_task_id ON download_state_transitions(download_task_id);

-- Media items table
CREATE TABLE IF NOT EXISTS media_items (
    id BIGSERIAL PRIMARY KEY,
    tmdb_id INTEGER,
    title VARCHAR(500) NOT NULL,
    year INTEGER,
    type VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_media_items_tmdb_id ON media_items(tmdb_id);
CREATE INDEX IF NOT EXISTS idx_media_items_type ON media_items(type);

-- Media files table
CREATE TABLE IF NOT EXISTS media_files (
    id BIGSERIAL PRIMARY KEY,
    media_item_id BIGINT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    file_path VARCHAR(1000) NOT NULL,
    container VARCHAR(50),
    codec VARCHAR(100),
    duration DOUBLE PRECISION,
    width INTEGER,
    height INTEGER,
    size BIGINT
);

CREATE INDEX IF NOT EXISTS idx_media_files_item_id ON media_files(media_item_id);