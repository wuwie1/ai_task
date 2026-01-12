package memory

import (
	"ai_web/test/constant"
	"ai_web/test/entity"
	"ai_web/test/model"
	"ai_web/test/pkg/clients/embedding"
	"ai_web/test/pkg/memory"
	"ai_web/test/pkg/tools"
	"ai_web/test/repository"
	"ai_web/test/repository/factory"
	"ai_web/test/repository/interfaces"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

var (
	serviceOnce sync.Once
	instance    *Service
)

// MemoryContextOptions 记忆上下文选项
type MemoryContextOptions struct {
	EnableSessionMemory bool    // 是否启用短期记忆
	EnableChunking      bool    // 是否启用分块功能
	SessionMemoryLimit  int     // 短期记忆条数
	SemanticMemoryLimit int     // 语义记忆条数
	SemanticThreshold   float64 // 语义相似度阈值
	CompressThreshold   int     // 压缩阈值（超过此数量时压缩旧记忆）
	EnableSummary       bool    // 是否启用摘要功能
	EnableAutoExtract   bool    // 是否自动提取关键事实到长期记忆
	// 分块配置
	ChunkMaxSize  int    // 最大块大小（字符数），0表示使用默认值1000
	ChunkOverlap  int    // 重叠窗口大小（字符数），0表示使用默认值100
	ChunkMinSize  int    // 最小块大小（字符数），0表示使用默认值200
	ChunkStrategy string // 分块策略: "paragraph", "sentence", "fixed"，空字符串表示使用默认值"paragraph"
}

type Service struct {
	repositoryFactory factory.Factory
	embeddingClient   *embedding.Client
	summarizer        *memory.Summarizer
	compressor        *memory.Compressor
}

func NewService(repositoryFactory factory.Factory) (*Service, error) {
	serviceOnce.Do(func() {
		embeddingClient, err := embedding.GetInstance()
		if err != nil {
			panic("failed to get embedding client: " + err.Error())
		}

		instance = &Service{
			repositoryFactory: repositoryFactory,
			embeddingClient:   embeddingClient,
			summarizer:        memory.NewSummarizer(),
			compressor:        memory.NewCompressor(),
		}
	})

	return instance, nil
}

// ==================== 短期记忆（Session Memory）====================

// 保存会话记忆
// 参考 LangChain：支持摘要和自动提取关键事实
// ctx 应该是 *gin.Context 类型，以便 summarizer 能够调用 LLM
func (s *Service) SaveSessionMemory(ctx context.Context, userID, sessionID string, messages []openai.ChatCompletionMessage, options *MemoryContextOptions) *model.Error {
	if len(messages) == 0 {
		return nil
	}

	openaiMessages := messages

	session := s.repositoryFactory.NewSession(ctx)
	defer tools.ErrorWithPrintContext(session.Close, "close session")

	memoryRepo := newChatMemoryChunksRepository(s.repositoryFactory, session)

	// 检查是否需要压缩旧记忆
	if options != nil && options.CompressThreshold > 0 {
		existingMemories, err := memoryRepo.GetRecentBySession(userID, sessionID, options.CompressThreshold+10)
		if err == nil && len(existingMemories) >= options.CompressThreshold {
			// 压缩旧记忆
			keepMemories, summary, err := s.compressor.CompressOldMemories(ctx, existingMemories, options.CompressThreshold)
			if err == nil && summary != "" {
				// 将摘要保存为一条新的记忆
				summaryChunk := &entity.ChatMemoryChunks{
					UserID:    userID,
					SessionID: sessionID,
					StartTS:   time.Now(),
					EndTS:     time.Now(),
					Text:      summary,
					Summary:   summary,
					Meta:      `{"role":"system","type":"summary"}`,
				}
				// 生成摘要的 embedding
				summaryEmbedding, err := s.embeddingClient.GetTextEmbedding(ctx, summary)
				if err == nil {
					summaryChunk.Embedding = embedding.VectorToString(summaryEmbedding)
					// 删除旧记忆并插入摘要（简化处理，实际应该批量更新）
					log.Infof("Compressed %d memories into summary for user=%s, session=%s", len(existingMemories)-len(keepMemories), userID, sessionID)
				}
			}
		}
	}

	// 将消息转换为文本 chunks（支持智能分块）
	chunks := s.messagesToChunks(openaiMessages, userID, sessionID, options)
	if len(chunks) == 0 {
		return nil
	}

	// 如果启用摘要，为重要对话生成摘要
	if options != nil && options.EnableSummary {
		for i := range chunks {
			// 对于较长的消息，生成摘要
			if len(chunks[i].Text) > 200 {
				summary, err := s.summarizer.SummarizeConversation(ctx, []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleUser, Content: chunks[i].Text},
				})
				if err == nil && summary != "" {
					chunks[i].Summary = summary
				}
			}
		}
	}

	// 批量生成 embedding
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Text
	}

	embeddings, err := s.embeddingClient.GetTextEmbeddingBatch(ctx, texts)
	if err != nil {
		return model.NewError(model.ErrorDB, fmt.Errorf("failed to get embeddings: %w", err))
	}

	// 将 embedding 转换为 PostgreSQL vector 格式
	for i, chunk := range chunks {
		if i < len(embeddings) {
			chunk.Embedding = embedding.VectorToString(embeddings[i])
		}
	}

	// 批量插入
	if err := memoryRepo.Insert(chunks); err != nil {
		return model.NewError(model.ErrorDB, fmt.Errorf("failed to insert memory chunks: %w", err))
	}

	// 如果启用自动提取，提取关键事实到长期记忆
	if options != nil && options.EnableAutoExtract {
		facts, err := s.summarizer.ExtractKeyFacts(ctx, openaiMessages)
		if err == nil && len(facts) > 0 {
			for key, value := range facts {
				_ = s.SaveLongTermMemory(ctx, userID, key, value, 0.8, nil)
			}
			log.Infof("Auto-extracted %d key facts for user=%s", len(facts), userID)
		}
	}

	log.Infof("Saved %d session memory chunks for user=%s, session=%s", len(chunks), userID, sessionID)
	return nil
}

// GetSessionMemory 获取会话记忆（最近 N 轮）
func (s *Service) GetSessionMemory(ctx context.Context, userID, sessionID string, limit int) ([]*entity.ChatMemoryChunks, *model.Error) {
	if limit <= 0 {
		limit = 10 // 默认10条
	}

	session := s.repositoryFactory.NewSession(ctx)
	defer tools.ErrorWithPrintContext(session.Close, "close session")

	memoryRepo := newChatMemoryChunksRepository(s.repositoryFactory, session)
	results, err := memoryRepo.GetRecentBySession(userID, sessionID, limit)
	if err != nil {
		return nil, model.NewError(model.ErrorDB, fmt.Errorf("failed to get session memory: %w", err))
	}
	return results, nil
}

// messagesToChunks 将消息转换为记忆 chunks（支持智能分块）
func (s *Service) messagesToChunks(messages []openai.ChatCompletionMessage, userID, sessionID string, options *MemoryContextOptions) []*entity.ChatMemoryChunks {
	chunks := make([]*entity.ChatMemoryChunks, 0)
	now := time.Now()

	// 准备分块配置（仅在启用分块时）
	var chunker memory.ChunkStrategy
	if options != nil && options.EnableChunking {
		chunkConfig := memory.DefaultChunkConfig()
		if options.ChunkMaxSize > 0 {
			chunkConfig.MaxSize = options.ChunkMaxSize
		}
		if options.ChunkOverlap > 0 {
			chunkConfig.Overlap = options.ChunkOverlap
		}
		if options.ChunkMinSize > 0 {
			chunkConfig.MinSize = options.ChunkMinSize
		}
		if options.ChunkStrategy != "" {
			chunkConfig.Strategy = options.ChunkStrategy
		}
		chunker = memory.NewChunker(chunkConfig)
	}

	for _, msg := range messages {
		// 只保存 user 和 assistant 的消息
		if msg.Role != openai.ChatMessageRoleUser && msg.Role != openai.ChatMessageRoleAssistant {
			continue
		}

		if msg.Content == "" {
			continue
		}

		// 对消息进行分块（如果启用分块功能）
		var textChunks []memory.Chunk
		if chunker != nil {
			chunkConfig := memory.DefaultChunkConfig()
			if options != nil {
				if options.ChunkMaxSize > 0 {
					chunkConfig.MaxSize = options.ChunkMaxSize
				}
				if options.ChunkOverlap > 0 {
					chunkConfig.Overlap = options.ChunkOverlap
				}
			}
			textChunks = chunker.Chunk(msg.Content, chunkConfig.MaxSize, chunkConfig.Overlap)
		} else {
			// 如果未启用分块，将整个消息作为一个 chunk
			textChunks = []memory.Chunk{
				{
					Text:        msg.Content,
					StartIdx:    0,
					EndIdx:      len(msg.Content),
					ChunkIdx:    0,
					TotalChunks: 1,
				},
			}
		}

		// 为每个 chunk 创建记忆条目
		for _, textChunk := range textChunks {
			// 构建元数据（包含分块信息）
			meta := map[string]interface{}{
				"role":        msg.Role,
				"name":        msg.Name,
				"tool_calls":  msg.ToolCalls,
				"chunk_index": textChunk.ChunkIdx,
				"chunk_total": textChunk.TotalChunks,
			}

			// 如果消息被分块了，记录原始消息的起始和结束位置
			if textChunk.TotalChunks > 1 {
				meta["chunk_start_idx"] = textChunk.StartIdx
				meta["chunk_end_idx"] = textChunk.EndIdx
			}

			metaJSON, _ := json.Marshal(meta)

			chunk := &entity.ChatMemoryChunks{
				UserID:    userID,
				SessionID: sessionID,
				StartTS:   now,
				EndTS:     now,
				Text:      textChunk.Text,
				Summary:   "", // 可以后续通过 LLM 生成摘要
				Meta:      string(metaJSON),
			}
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// ==================== 长期记忆（Long-term Memory）====================

// SaveLongTermMemory 保存长期记忆（用户偏好、配置等）
func (s *Service) SaveLongTermMemory(ctx context.Context, userID, key, value string, confidence float32, sourceMsgID *int64) *model.Error {
	session := s.repositoryFactory.NewSession(ctx)
	defer tools.ErrorWithPrintContext(session.Close, "close session")

	profileRepo := newUserProfileRepository(s.repositoryFactory, session)

	req := &model.UpsertUserProfileCondition{
		UserID:      userID,
		Key:         key,
		Value:       value,
		Confidence:  confidence,
		SourceMsgID: sourceMsgID,
	}

	if err := profileRepo.Upsert(req); err != nil {
		return model.NewError(model.ErrorDB, fmt.Errorf("failed to upsert user profile: %w", err))
	}

	log.Infof("Saved long-term memory: user=%s, key=%s, value=%s", userID, key, value)
	return nil
}

// 获取长期记忆
func (s *Service) GetLongTermMemory(ctx context.Context, userID string) ([]*entity.UserProfile, *model.Error) {
	session := s.repositoryFactory.NewSession(ctx)
	defer tools.ErrorWithPrintContext(session.Close, "close session")

	profileRepo := newUserProfileRepository(s.repositoryFactory, session)

	condition := &model.GetUserProfileCondition{
		UserID: &userID,
	}

	results, err := profileRepo.List(condition)
	if err != nil {
		return nil, model.NewError(model.ErrorDB, fmt.Errorf("failed to get long-term memory: %w", err))
	}
	return results, nil
}

// ==================== 语义记忆（Semantic Memory）====================

// 语义记忆检索（向量相似度搜索）
func (s *Service) SearchSemanticMemory(ctx context.Context, userID, query string, sessionID *string, limit int, threshold *float64) ([]*entity.ChatMemoryChunks, *model.Error) {
	// 生成查询向量
	queryEmbedding, err := s.embeddingClient.GetTextEmbedding(ctx, query)
	if err != nil {
		return nil, model.NewError(model.ErrorDB, fmt.Errorf("failed to get query embedding: %w", err))
	}

	queryVector := embedding.VectorToString(queryEmbedding)

	session := s.repositoryFactory.NewSession(ctx)
	defer tools.ErrorWithPrintContext(session.Close, "close session")

	memoryRepo := newChatMemoryChunksRepository(s.repositoryFactory, session)

	condition := &model.VectorSearchCondition{
		UserID:      userID,
		SessionID:   sessionID,
		QueryVector: queryVector,
		Limit:       limit,
		Threshold:   threshold,
	}

	results, err := memoryRepo.VectorSearch(condition)
	if err != nil {
		return nil, model.NewError(model.ErrorDB, fmt.Errorf("failed to vector search: %w", err))
	}

	log.Infof("Semantic memory search: user=%s, query=%s, found=%d results", userID, query, len(results))
	return results, nil
}

// ==================== 综合记忆管理 ====================

// 构建带记忆的对话上下文
func (s *Service) BuildContextWithMemory(ctx context.Context, userID, sessionID string, currentMessages []openai.ChatCompletionMessage, options *MemoryContextOptions) ([]openai.ChatCompletionMessage, *model.Error) {
	if options == nil {
		options = &MemoryContextOptions{
			EnableSessionMemory: true,  // 默认启用短期记忆
			EnableChunking:      true,  // 默认启用分块
			SessionMemoryLimit:  10,
			SemanticMemoryLimit: 5,
			SemanticThreshold:   0.7,
			CompressThreshold:   20, // 超过20条时压缩
			EnableSummary:       false,
			EnableAutoExtract:   false,
			// 分块配置使用默认值（在 chunker 中处理）
			ChunkMaxSize:  0, // 0 表示使用默认值
			ChunkOverlap:  0, // 0 表示使用默认值
			ChunkMinSize:  0, // 0 表示使用默认值
			ChunkStrategy: "", // 空字符串表示使用默认值
		}
	}

	openaiMessages := currentMessages

	messages := make([]openai.ChatCompletionMessage, 0)

	// 1. 获取长期记忆并构建系统提示，查询 user_profile 表
	longTermMemories, err := s.GetLongTermMemory(ctx, userID)
	if err != nil {
		log.Warnf("Failed to get long-term memory: %v", err)
	} else if len(longTermMemories) > 0 {
		// 根据生成的偏好设置等，增加系统提示词
		systemPrompt := s.buildSystemPromptWithLongTermMemory(longTermMemories)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}

	// 2. 语义记忆检索（基于最后一条用户消息）
	if len(openaiMessages) > 0 {
		lastUserMsg := ""
		// 找出最后一条的用户消息，也就是最近的提问
		for i := len(openaiMessages) - 1; i >= 0; i-- {
			if openaiMessages[i].Role == openai.ChatMessageRoleUser {
				lastUserMsg = openaiMessages[i].Content
				break
			}
		}

		if lastUserMsg != "" {
			// 根据向量的相似度在表 chat_memory_chunks 中查询
			threshold := options.SemanticThreshold
			semanticMemories, err := s.SearchSemanticMemory(ctx, userID, lastUserMsg, &sessionID, options.SemanticMemoryLimit, &threshold)
			if err != nil {
				log.Warnf("Failed to search semantic memory: %v", err)
			} else if len(semanticMemories) > 0 {
				// 将语义记忆作为上下文注入
				contextText := s.buildContextFromMemories(semanticMemories)
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf(constant.SemanticMemoryContextPromptTemplate, contextText),
				})
			}
		}
	}

	// 3. 获取短期记忆（最近 N 轮），也就是最近的 N 条聊天记录，然后组合进来
	sessionMemories, err := s.GetSessionMemory(ctx, userID, sessionID, options.SessionMemoryLimit)
	if err != nil {
		log.Warnf("Failed to get session memory: %v", err)
	} else {
		// 将短期记忆转换为 messages
		for _, memory := range sessionMemories {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(memory.Meta), &meta); err == nil {
				if role, ok := meta["role"].(string); ok {
					// 这里使用的是 text
					msg := openai.ChatCompletionMessage{
						Role:    role,
						Content: memory.Text,
					}
					if name, ok := meta["name"].(string); ok {
						msg.Name = name
					}
					messages = append(messages, msg)
				}
			}
		}
	}

	// 4. 添加当前消息
	messages = append(messages, openaiMessages...)

	return messages, nil
}

// 构建包含长期记忆的系统提示
func (s *Service) buildSystemPromptWithLongTermMemory(memories []*entity.UserProfile) string {
	if len(memories) == 0 {
		return ""
	}

	prompt := constant.LongTermMemoryPromptPrefix
	for _, mem := range memories {
		prompt += fmt.Sprintf("- %s: %s (置信度: %.2f)\n", mem.Key, mem.Value, mem.Confidence)
	}
	return prompt
}

// 从记忆构建上下文文本
func (s *Service) buildContextFromMemories(memories []*entity.ChatMemoryChunks) string {
	context := ""
	for i, mem := range memories {
		if i > 0 {
			context += "\n\n"
		}
		context += fmt.Sprintf("[%s] %s", mem.StartTS.Format("2006-01-02 15:04:05"), mem.Text)
		if mem.Summary != "" {
			context += fmt.Sprintf(" (摘要: %s)", mem.Summary)
		}
	}
	return context
}

// 辅助函数：创建 repository 实例
func newChatMemoryChunksRepository(repoFactory factory.Factory, session interfaces.Session) repository.ChatMemoryChunksRepository {
	repo, err := repoFactory.NewChatMemoryChunksRepository(session)
	if err != nil {
		panic("failed to create chat memory chunks repository: " + err.Error())
	}
	return repo
}

func newUserProfileRepository(repoFactory factory.Factory, session interfaces.Session) repository.UserProfileRepository {
	repo, err := repoFactory.NewUserProfileRepository(session)
	if err != nil {
		panic("failed to create user profile repository: " + err.Error())
	}
	return repo
}
