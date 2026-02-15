-- +goose Up
CREATE TABLE channels (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,
    description  TEXT,
    type         VARCHAR(20) NOT NULL DEFAULT 'public',
    is_encrypted BOOLEAN DEFAULT false,
    created_by   UUID NOT NULL REFERENCES users(id),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    archived_at  TIMESTAMPTZ,
    UNIQUE(workspace_id, name)
);

CREATE TABLE channel_members (
    channel_id       UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role             VARCHAR(20) DEFAULT 'member',
    encrypted_key    BYTEA,
    last_read_msg_id UUID,
    joined_at        TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (channel_id, user_id)
);

-- +goose Down
DROP TABLE channel_members;
DROP TABLE channels;
