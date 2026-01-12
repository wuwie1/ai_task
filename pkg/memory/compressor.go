package memory

import (
	"ai_web/test/entity"
	"context"
	"encoding/json"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

// Compressor 记忆压缩器
// 参考 LangChain：当对话历史过长时，自动压缩旧记忆
type Compressor struct {
	summarizer *Summarizer
}

// NewCompressor 创建记忆压缩器
func NewCompressor() *Compressor {
	return &Compressor{
		summarizer: NewSummarizer(),
	}
}

// 压缩旧的会话记忆
// 当会话记忆超过阈值时，将旧记忆压缩为摘要
func (c *Compressor) CompressOldMemories(ctx context.Context, memories []*entity.ChatMemoryChunks, maxKeep int) ([]*entity.ChatMemoryChunks, string, error) {
	if len(memories) <= maxKeep {
		return memories, "", nil
	}

	// 保留最新的 maxKeep 条记忆
	keepMemories := memories[len(memories)-maxKeep:]
	compressMemories := memories[:len(memories)-maxKeep]

	// 将需要压缩的记忆转换为对话格式并生成摘要
	summaryText := c.buildMemoryText(compressMemories)

	// 使用 summarizer 生成摘要
	summary, err := c.summarizer.SummarizeConversation(ctx, c.memoriesToMessages(compressMemories))
	if err != nil {
		log.Warnf("Failed to compress memories: %v", err)
		// 如果摘要失败，使用简单的文本摘要
		summary = c.simpleSummary(summaryText)
	}

	return keepMemories, summary, nil
}

// 将记忆转换为消息格式
func (c *Compressor) memoriesToMessages(memories []*entity.ChatMemoryChunks) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, len(memories))
	for _, mem := range memories {
		var meta map[string]interface{}
		if err := json.Unmarshal([]byte(mem.Meta), &meta); err == nil {
			role := openai.ChatMessageRoleUser
			if r, ok := meta["role"].(string); ok {
				role = r
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    role,
				Content: mem.Text,
			})
		} else {
			// 如果解析失败，默认作为用户消息
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: mem.Text,
			})
		}
	}
	return messages
}

// 构建记忆文本
func (c *Compressor) buildMemoryText(memories []*entity.ChatMemoryChunks) string {
	text := ""
	for _, mem := range memories {
		if text != "" {
			text += "\n"
		}
		text += mem.Text
	}
	return text
}

// simpleSummary 简单的文本摘要（备用方案）
func (c *Compressor) simpleSummary(text string) string {
	if len(text) > 200 {
		return text[:200] + "..."
	}
	return text
}

// ShouldCompress 判断是否需要压缩
func (c *Compressor) ShouldCompress(memoryCount int, threshold int) bool {
	return memoryCount > threshold
}
