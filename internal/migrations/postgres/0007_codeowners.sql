-- SPDX-License-Identifier: AGPL-3.0
-- CODEOWNERS review requirement on branch protection (SQLite)

ALTER TABLE branch_protections ADD COLUMN require_codeowners_review BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE branch_protection_rules ADD COLUMN require_codeowners_review BOOLEAN NOT NULL DEFAULT false;
