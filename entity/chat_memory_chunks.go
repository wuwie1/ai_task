package entity

import "time"

const (
	TableNameChatMemoryChunks = "chat_memory_chunks"

	ChatMemoryChunksFieldID        = "id"
	ChatMemoryChunksFieldUserID    = "user_id"
	ChatMemoryChunksFieldSessionID = "session_id"
	ChatMemoryChunksFieldStartTS   = "start_ts"
	ChatMemoryChunksFieldEndTS     = "end_ts"
	ChatMemoryChunksFieldText      = "text"
	ChatMemoryChunksFieldSummary   = "summary"
	ChatMemoryChunksFieldEmbedding = "embedding"
	ChatMemoryChunksFieldMeta      = "meta"
)

type ChatMemoryChunks struct {
	ID        int64     `xorm:"pk autoincr id" json:"id"`
	UserID    string    `xorm:"user_id" json:"user_id"`
	SessionID string    `xorm:"session_id" json:"session_id"`
	StartTS   time.Time `xorm:"start_ts" json:"start_ts"`
	EndTS     time.Time `xorm:"end_ts" json:"end_ts"`
	Text      string    `xorm:"text" json:"text"`
	Summary   string    `xorm:"summary" json:"summary"`
	Embedding string    `xorm:"embedding" json:"embedding"` // PostgreSQL vector 类型，存储为字符串
	Meta      string    `xorm:"meta" json:"meta"`           // JSONB 类型，存储为 JSON 字符串
}

func (e *ChatMemoryChunks) TableName() string {
	return TableNameChatMemoryChunks
}
