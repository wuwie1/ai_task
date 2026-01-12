package model

import "time"

// GetChatMemoryChunksCondition 查询条件（带分页和排序）
type GetChatMemoryChunksCondition struct {
	UserID    *string    `json:"user_id"`
	SessionID *string    `json:"session_id"`
	StartTS   *time.Time `json:"start_ts"`
	EndTS     *time.Time `json:"end_ts"`
	*Pager
	*Order
}

func (g *GetChatMemoryChunksCondition) GetPager() *Pager {
	return g.Pager
}

func (g *GetChatMemoryChunksCondition) GetOrder() *Order {
	return g.Order
}

// GetChatMemoryChunksConditionNoCount 查询条件（不带分页计数）
type GetChatMemoryChunksConditionNoCount struct {
	UserID    *string    `json:"user_id"`
	SessionID *string    `json:"session_id"`
	StartTS   *time.Time `json:"start_ts"`
	EndTS     *time.Time `json:"end_ts"`
}

// GetChatMemoryChunksConditionCount 仅计数查询条件
type GetChatMemoryChunksConditionCount struct {
	UserID    *string `json:"user_id"`
	SessionID *string `json:"session_id"`
}

// UpdateChatMemoryChunksCondition 更新条件
type UpdateChatMemoryChunksCondition struct {
	Text      *string `json:"text"`
	Summary   *string `json:"summary"`
	Embedding *string `json:"embedding"`
	Meta      *string `json:"meta"`
}

// VectorSearchCondition 向量检索条件
type VectorSearchCondition struct {
	UserID      string    `json:"user_id"`
	SessionID   *string   `json:"session_id"` // 可选，如果为空则跨会话检索
	QueryVector string    `json:"query_vector"` // 查询向量（字符串格式）
	Limit       int       `json:"limit"`         // 返回数量
	Threshold   *float64  `json:"threshold"`    // 相似度阈值（可选）
	StartTS     *time.Time `json:"start_ts"`     // 时间范围过滤（可选）
	EndTS       *time.Time `json:"end_ts"`       // 时间范围过滤（可选）
}

