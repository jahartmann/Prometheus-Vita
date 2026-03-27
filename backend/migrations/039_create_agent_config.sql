CREATE TABLE IF NOT EXISTS agent_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(100) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO agent_config (id, key, value) VALUES
    (gen_random_uuid(), 'llm_provider', 'ollama'),
    (gen_random_uuid(), 'llm_model', 'llama3'),
    (gen_random_uuid(), 'ollama_url', 'http://localhost:11434'),
    (gen_random_uuid(), 'openai_key', ''),
    (gen_random_uuid(), 'anthropic_key', '')
ON CONFLICT (key) DO NOTHING;
