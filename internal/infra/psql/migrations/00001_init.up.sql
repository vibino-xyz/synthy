CREATE TABLE IF NOT EXISTS repository (
    id varchar(255),
    full_name varchar(255),
    default_branch varchar(255),
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp,

    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS file (
    id varchar(255),
    repository_id varchar(255),
    path varchar(255) NOT NULL,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp,

    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE
);

CREATE TYPE symbol_kind AS ENUM ('FUNCTION', 'METHOD', 'CLASS', 'STRUCT', 'INTERFACE', 'ENUM');

CREATE TABLE IF NOT EXISTS symbol (
    id varchar(255),
    repository_id varchar(255),
    file_id varchar(255),
    kind symbol_kind NOT NULL,
    name varchar(255) NOT NULL,
    fq_name varchar(255) NOT NULL,
    receiver_type TEXT,
    start_line int,
    end_line int,
    code_object_uri TEXT,      -- raw source storage
    ast_object_uri TEXT,       -- serialized AST location
    summary_object_uri TEXT,   -- optional LLM summary
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp,

    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (file_id) REFERENCES file(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS call_edge (
    id varchar(255),
    repository_id varchar(255),
    caller_symbol_id varchar(255),
    callee_symbol_id varchar(255),
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp,

    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (caller_symbol_id) REFERENCES symbol(id) ON DELETE CASCADE,
    FOREIGN KEY (callee_symbol_id) REFERENCES symbol(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ownership_edge (
    id varchar(255),
    repository_id varchar(255),
    owner_symbol_id varchar(255),
    child_symbol_id varchar(255),
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp,

    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (owner_symbol_id) REFERENCES symbol(id) ON DELETE CASCADE,
    FOREIGN KEY (child_symbol_id) REFERENCES symbol(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS import_edge (
    id varchar(255),
    repository_id varchar(255),
    importer_symbol_id varchar(255),
    imported_symbol_id varchar(255),
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp,

    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (importer_symbol_id) REFERENCES symbol(id) ON DELETE CASCADE,
    FOREIGN KEY (imported_symbol_id) REFERENCES symbol(id) ON DELETE CASCADE
);

CREATE TYPE chunk_type AS ENUM ('FUNCTION', 'BLOCK', 'CLASS');

CREATE TABLE IF NOT EXISTS chunk (
    id varchar(255),
    repository_id varchar(255),
    symbol_id varchar(255),
    type chunk_type NOT NULL,
    start_line int,
    end_line int,
    token_count int,
    code_object_uri TEXT,      -- raw source storage
    ast_object_uri TEXT,       -- serialized AST location
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp,

    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (symbol_id) REFERENCES symbol(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS symbol_summary (
    id varchar(255),
    repository_id varchar(255),
    symbol_id varchar(255),
    summary TEXT,
    embedding_id TEXT,
    created_at timestamp default current_timestamp,
    updated_at timestamp default current_timestamp,

    PRIMARY KEY (id),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (symbol_id) REFERENCES symbol(id) ON DELETE CASCADE
);