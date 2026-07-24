-- GitHub App installations, one per organization. Created when an org admin
-- installs the Vibino GitHub App and synthy receives the installation_id.
--
-- organization_id references an organization owned by nexy (it arrives on the
-- caller's JWT). It is deliberately NOT a foreign key: synthy does not own the
-- organization table, so it stores the id as an external reference only.
CREATE TABLE IF NOT EXISTS github_installation (
    id VARCHAR(36),
    organization_id VARCHAR(36) NOT NULL,
    installation_id BIGINT NOT NULL,
    account_login TEXT NOT NULL,
    account_type TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (organization_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_github_installation_installation_id
    ON github_installation (installation_id);
