ALTER TABLE activities ADD COLUMN summary TEXT NOT NULL DEFAULT '';
ALTER TABLE activities ADD COLUMN repo_name VARCHAR(255) NOT NULL DEFAULT '';
CREATE INDEX idx_activities_actor ON activities (actor_id, id);
CREATE INDEX idx_activities_repo ON activities (repo_id, id);
