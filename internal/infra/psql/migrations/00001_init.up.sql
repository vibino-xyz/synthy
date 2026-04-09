-- +goose Up
-- +goose StatementBegin
CREATE TYPE organization_member_role as ENUM ('ADMIN', 'MEMBER');

CREATE TYPE repository_member_role as ENUM ('ADMIN', 'VIEWER');

CREATE TYPE repository_provider as ENUM ('GITHUB', 'GITLAB', 'OTHER');

CREATE TYPE language as ENUM ('GO', 'OTHER');

CREATE TYPE symbol_type as ENUM (
    'FUNCTION',
    'TYPE',
    'VARIABLE',
    'CONSTANT',
    'PACKAGE',
    'FIELD',
    'INTERFACE',
    'IMPORT',
    'OTHER'
);

CREATE TYPE receiver_type as ENUM ('VALUE', 'POINTER');

CREATE TYPE symbol_visibility as ENUM ('PUBLIC', 'PRIVATE');

CREATE TYPE chunk_type as ENUM ('FUNCTION', 'TYPE', 'BLOCK', 'OTHER');

CREATE TABLE IF NOT EXISTS "user" (
    id VARCHAR(32),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    profile_url TEXT,
    is_verified BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS organization (
    id VARCHAR(32),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    owner_id VARCHAR(32) NOT NULL,
    description TEXT,
    profile_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (owner_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS organization_member (
    id VARCHAR(32),
    organization_id VARCHAR(32) NOT NULL,
    user_id VARCHAR(32) NOT NULL,
    role organization_member_role NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (organization_id, user_id),
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS repository (
    id VARCHAR(32),
    organization_id VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    default_branch VARCHAR(255) NOT NULL,
    repository_url TEXT NOT NULL,
    storage_url TEXT NOT NULL,
    provider repository_provider NOT NULL,
    external_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (organization_id) REFERENCES organization(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS repository_member (
    id VARCHAR(32),
    repository_id VARCHAR(32) NOT NULL,
    user_id VARCHAR(32) NOT NULL,
    role repository_member_role NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (repository_id, user_id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS file (
    id VARCHAR(32),
    repository_id VARCHAR(32) NOT NULL,
    path TEXT NOT NULL,
    name TEXT NOT NULL,
    extension VARCHAR(30) NOT NULL,
    checksum VARCHAR(128) NOT NULL,
    language language NOT NULL,
    is_binary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (repository_id, path),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS symbol (
    id VARCHAR(32),
    file_id VARCHAR(32) NOT NULL,
    repository_id VARCHAR(32) NOT NULL,
    fq_name TEXT NOT NULL,
    type symbol_type NOT NULL,
    language language NOT NULL,
    signature TEXT NOT NULL,
    receiver_type receiver_type,
    visibility symbol_visibility NOT NULL,
    parent_symbol_id VARCHAR(32),
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    start_column INTEGER NOT NULL,
    end_column INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (file_id) REFERENCES file(id) ON DELETE CASCADE,
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_symbol_id) REFERENCES symbol(id) ON DELETE
    SET
        NULL
);

CREATE TABLE IF NOT EXISTS call_edge (
    id VARCHAR(32),
    repository_id VARCHAR(32) NOT NULL,
    caller_symbol_id VARCHAR(32) NOT NULL,
    callee_symbol_id VARCHAR(32) NOT NULL,
    file_id VARCHAR(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (caller_symbol_id) REFERENCES symbol(id) ON DELETE CASCADE,
    FOREIGN KEY (callee_symbol_id) REFERENCES symbol(id) ON DELETE CASCADE,
    FOREIGN KEY (file_id) REFERENCES file(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS import_edge (
    id VARCHAR(32),
    repository_id VARCHAR(32) NOT NULL,
    from_file_id VARCHAR(32) NOT NULL,
    to_file_id VARCHAR(32) NOT NULL,
    import_path TEXT NOT NULL,
    is_external BOOLEAN DEFAULT FALSE,
    alias TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (from_file_id) REFERENCES file(id) ON DELETE CASCADE,
    FOREIGN KEY (to_file_id) REFERENCES file(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS code_chunk (
    id VARCHAR(32),
    repository_id VARCHAR(32) NOT NULL,
    file_id VARCHAR(32) NOT NULL,
    symbol_id VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    content_hash VARCHAR(128) NOT NULL,
    type chunk_type NOT NULL,
    language language NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    embedding_id VARCHAR(255),
    summary TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (file_id) REFERENCES file(id) ON DELETE CASCADE,
    FOREIGN KEY (symbol_id) REFERENCES symbol(id) ON DELETE CASCADE
);

-- Indexes (important for retrieval performance)
CREATE INDEX IF NOT EXISTS idx_repo_org ON repository(organization_id);

CREATE INDEX IF NOT EXISTS idx_file_repo ON file(repository_id);

CREATE INDEX IF NOT EXISTS idx_symbol_file ON symbol(file_id);

CREATE INDEX IF NOT EXISTS idx_symbol_repo ON symbol(repository_id);

CREATE INDEX IF NOT EXISTS idx_symbol_parent ON symbol(parent_symbol_id);

CREATE INDEX IF NOT EXISTS idx_calledge_repo ON call_edge(repository_id);

CREATE INDEX IF NOT EXISTS idx_calledge_caller ON call_edge(caller_symbol_id);

CREATE INDEX IF NOT EXISTS idx_calledge_callee ON call_edge(callee_symbol_id);

CREATE INDEX IF NOT EXISTS idx_importedge_repo ON import_edge(repository_id);

CREATE INDEX IF NOT EXISTS idx_importedge_from ON import_edge(from_file_id);

CREATE INDEX IF NOT EXISTS idx_importedge_to ON import_edge(to_file_id);

CREATE INDEX IF NOT EXISTS idx_chunk_repo ON code_chunk(repository_id);

CREATE INDEX IF NOT EXISTS idx_chunk_file ON code_chunk(file_id);

CREATE INDEX IF NOT EXISTS idx_chunk_symbol ON code_chunk(symbol_id);

-- +goose StatementEnd