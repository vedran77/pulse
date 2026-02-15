-- +goose Up
CREATE TABLE messages (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id        UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    sender_id         UUID NOT NULL REFERENCES users(id),
    content           TEXT,
    content_encrypted BYTEA,
    nonce             BYTEA,
    type              VARCHAR(20) DEFAULT 'text',
    parent_id         UUID REFERENCES messages(id),
    edited_at         TIMESTAMPTZ,
    deleted_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_channel_created ON messages(channel_id, created_at DESC);
CREATE INDEX idx_messages_parent ON messages(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_messages_sender ON messages(sender_id);

-- +goose Down
DROP TABLE messages;
