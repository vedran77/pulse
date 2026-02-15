-- +goose Up
CREATE TABLE pulsemate_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    CHECK (sender_id != receiver_id),
    CHECK (status IN ('pending', 'rejected')),
    UNIQUE(sender_id, receiver_id)
);
CREATE INDEX idx_pulsemate_requests_receiver ON pulsemate_requests(receiver_id, status);
CREATE INDEX idx_pulsemate_requests_sender ON pulsemate_requests(sender_id, status);

CREATE TABLE pulsemates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user1_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CHECK (user1_id < user2_id),
    UNIQUE(user1_id, user2_id)
);
CREATE INDEX idx_pulsemates_user1 ON pulsemates(user1_id);
CREATE INDEX idx_pulsemates_user2 ON pulsemates(user2_id);

-- +goose Down
DROP TABLE pulsemates;
DROP TABLE pulsemate_requests;
