package model

import "github.com/sashabaranov/go-openai"

// ChatRequest 聊天请求
type ChatRequest struct {
	UserID    string                        `json:"user_id" binding:"required"`
	SessionID string                        `json:"session_id" binding:"required"`
	Messages  []openai.ChatCompletionMessage `json:"messages" binding:"required"`
	Stream   bool                           `json:"stream"` // 是否流式返回
}

// ChatResponse 聊天响应（非流式）
type ChatResponse struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
}

// MemoryContextOptionsRequest 记忆上下文选项（可选）
// 参考 LangChain：支持摘要、压缩和自动提取
type MemoryContextOptionsRequest struct {
	EnableSessionMemory *bool    `json:"enable_session_memory"` // 是否启用短期记忆
	EnableChunking      *bool    `json:"enable_chunking"`        // 是否启用分块功能
	SessionMemoryLimit  *int     `json:"session_memory_limit"`  // 短期记忆条数
	SemanticMemoryLimit *int     `json:"semantic_memory_limit"` // 语义记忆条数
	SemanticThreshold   *float64 `json:"semantic_threshold"`   // 语义相似度阈值
	CompressThreshold   *int     `json:"compress_threshold"`   // 压缩阈值（超过此数量时压缩旧记忆）
	EnableSummary       *bool    `json:"enable_summary"`       // 是否启用摘要功能
	EnableAutoExtract   *bool    `json:"enable_auto_extract"`  // 是否自动提取关键事实到长期记忆
	// 分块配置
	ChunkMaxSize  *int    `json:"chunk_max_size"`  // 最大块大小（字符数），0或nil表示使用默认值1000
	ChunkOverlap  *int    `json:"chunk_overlap"`   // 重叠窗口大小（字符数），0或nil表示使用默认值100
	ChunkMinSize  *int    `json:"chunk_min_size"`  // 最小块大小（字符数），0或nil表示使用默认值200
	ChunkStrategy *string `json:"chunk_strategy"`   // 分块策略: "paragraph", "sentence", "fixed"，空字符串或nil表示使用默认值"paragraph"
}

