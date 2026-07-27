-- organization_id on repository references an organization owned by nexy (it
-- arrives on the caller's JWT / ingestion message). synthy does not own the
-- organization table, so drop the foreign key and treat it as an external
-- reference, matching github_installation.
ALTER TABLE repository DROP CONSTRAINT IF EXISTS repository_organization_id_fkey;
