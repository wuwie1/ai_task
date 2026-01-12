-- 创建聊天记忆表
CREATE TABLE IF NOT EXISTS chat_memory_chunks (
  id          bigserial PRIMARY KEY,
  user_id     text NOT NULL,
  session_id  text NOT NULL,
  start_ts    timestamptz NOT NULL,
  end_ts      timestamptz NOT NULL,
  text        text NOT NULL,
  summary     text,
  embedding   vector(1536) NOT NULL,
  meta        jsonb NOT NULL DEFAULT '{}'::jsonb
);

-- 向量索引（HNSW，余弦）
CREATE INDEX IF NOT EXISTS chat_memory_hnsw
ON chat_memory_chunks USING hnsw (embedding vector_cosine_ops);

-- 常规索引（过滤 + 时间范围）
CREATE INDEX IF NOT EXISTS chat_memory_session_time
ON chat_memory_chunks (user_id, session_id, start_ts);

-- 创建用户画像表（长期记忆）
CREATE TABLE IF NOT EXISTS user_profile (
  id            bigserial PRIMARY KEY,
  user_id       text NOT NULL,
  key           text NOT NULL,
  value         text NOT NULL,
  confidence    float4 NOT NULL DEFAULT 1.0,
  updated_at    timestamptz NOT NULL DEFAULT now(),
  source_msg_id bigint,
  meta          jsonb NOT NULL DEFAULT '{}'::jsonb,
  UNIQUE(user_id, key)
);

-- 用户画像索引
CREATE INDEX IF NOT EXISTS user_profile_user_id
ON user_profile (user_id);

