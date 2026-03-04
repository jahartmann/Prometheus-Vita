-- +migrate up
CREATE TABLE telegram_user_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    telegram_chat_id BIGINT UNIQUE,
    telegram_username VARCHAR(255),
    verification_code VARCHAR(6),
    is_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMPTZ
);

CREATE INDEX idx_telegram_user_links_user_id ON telegram_user_links(user_id);
CREATE INDEX idx_telegram_user_links_chat_id ON telegram_user_links(telegram_chat_id);

CREATE TABLE telegram_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_chat_id BIGINT NOT NULL,
    conversation_id UUID REFERENCES chat_conversations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_telegram_conversations_chat_id ON telegram_conversations(telegram_chat_id);
