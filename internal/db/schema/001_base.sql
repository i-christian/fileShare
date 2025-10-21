-- +goose Up

-- Types
CREATE TYPE upload_status AS ENUM ('pending', 'completed', 'failed');
CREATE TYPE user_role AS ENUM ('admin', 'user');
CREATE TYPE file_visibility AS ENUM ('public', 'private');

-- Users table: Stores user account information
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid7(),
    last_name VARCHAR(50) NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    is_verified BOOLEAN DEFAULT FALSE,
    role user_role NOT NULL DEFAULT 'user',
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

--  Email verification tokens
CREATE TABLE email_verification_tokens (
    token_id    UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token       VARCHAR(255) UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    used        BOOLEAN NOT NULL DEFAULT FALSE
);

--  Password reset tokens
CREATE TABLE password_reset_tokens (
    token_id    UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token       VARCHAR(255) UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    used        BOOLEAN NOT NULL DEFAULT FALSE
);

-- API Keys table: Allows users to generate keys for programmatic access
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    prefix VARCHAR(16) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

-- Files table: Stores metadata
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    storage_key TEXT NOT NULL UNIQUE,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    visibility file_visibility NOT NULL DEFAULT 'private',
    thumbnail_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Share Links table: Manages secure, time-sensitive, and protected links
CREATE TABLE share_links (
    id UUID PRIMARY KEY DEFAULT uuid7(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    password_hash VARCHAR(255),
    expires_at TIMESTAMPTZ,
    download_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Upload Sessions table for chunked uploads
CREATE TABLE upload_sessions (
    id UUID PRIMARY KEY DEFAULT uuid7(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    total_chunks INT NOT NULL,
    uploaded_chunks INT DEFAULT 0,
    status upload_status DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_files_user_id ON files(user_id);
CREATE INDEX idx_files_visibility ON files(visibility);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_share_links_file_id ON share_links(file_id);
CREATE INDEX idx_share_links_token ON share_links(token);
CREATE INDEX idx_upload_sessions_user_id ON upload_sessions(user_id);


-- +goose Down
-- Drop tables
DROP TABLE IF EXISTS upload_sessions;
DROP TABLE IF EXISTS share_links;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS users;

-- Drop types
DROP TYPE IF EXISTS file_visibility;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS upload_status;
