-- SPDX-License-Identifier: AGPL-3.0
-- gitli runner schema (SQLite)

CREATE TABLE runners (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    runner_id VARCHAR(128) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    labels TEXT NOT NULL,
    version VARCHAR(64) NOT NULL DEFAULT '',
    os VARCHAR(64) NOT NULL DEFAULT '',
    arch VARCHAR(64) NOT NULL DEFAULT '',
    last_seen_at DATETIME,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE runner_jobs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    runner_id VARCHAR(128),
    repo_id BIGINT NOT NULL,
    commit_sha VARCHAR(64) NOT NULL,
    ref VARCHAR(255) NOT NULL,
    workflow_path VARCHAR(512) NOT NULL,
    workflow_content TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    started_at DATETIME,
    finished_at DATETIME,
    exit_code BIGINT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE runner_job_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id BIGINT NOT NULL,
    chunk TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_runner_jobs_status ON runner_jobs (status);
CREATE INDEX idx_runner_jobs_runner ON runner_jobs (runner_id);
CREATE INDEX idx_runner_jobs_repo ON runner_jobs (repo_id);
CREATE INDEX idx_runner_job_logs_job ON runner_job_logs (job_id);
