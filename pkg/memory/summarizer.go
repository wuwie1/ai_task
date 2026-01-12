package memory

import (
	"ai_web/test/constant"
	"ai_web/test/pkg/clients/llm_model"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

// Summarizer 摘要生成器
type Summarizer struct {
	llmClient *llm_model.ClientChatModel
}

// NewSummarizer 创建摘要生成器
func NewSummarizer() *Summarizer {
	return &Summarizer{
		llmClient: llm_model.GetInstance(),
	}
}

// 对对话进行摘要
// 参考 LangChain 的摘要机制：将多轮对话压缩为简洁的摘要
func (s *Summarizer) SummarizeConversation(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// 构建对话文本
	conversationText := s.buildConversationText(messages)

	// 构建摘要提示
	summaryPrompt := fmt.Sprintf(constant.SummaryUserPromptTemplate, conversationText)

	summaryMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: constant.SummarySystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: summaryPrompt,
		},
	}

	summary, err := s.llmClient.PostChatCompletionsNonStreamContent(ctx, summaryMessages)
	if err != nil {
		log.Warnf("Failed to generate summary: %v", err)
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return strings.TrimSpace(summary), nil
}

// 从对话中提取关键事实（用于长期记忆）
// 参考 LangChain 的记忆整合模式：让 LLM 决定如何扩展或整合记忆状态
func (s *Summarizer) ExtractKeyFacts(ctx context.Context, messages []openai.ChatCompletionMessage) (map[string]string, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	conversationText := s.buildConversationText(messages)

	extractPrompt := fmt.Sprintf(constant.ExtractKeyFactsUserPromptTemplate, conversationText)

	extractMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: constant.ExtractKeyFactsSystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: extractPrompt,
		},
	}

	result, err := s.llmClient.PostChatCompletionsNonStreamContent(ctx, extractMessages)
	if err != nil {
		log.Warnf("Failed to extract key facts: %v", err)
		return nil, fmt.Errorf("failed to extract key facts: %w", err)
	}

	// 解析 JSON 结果
	facts := make(map[string]string)
	if result != "" && result != "{}" {
		// 清理响应内容，移除可能的 markdown 代码块标记
		cleanedResult := cleanJSONResponse(result)

		// 尝试解析 JSON
		var parsedFacts map[string]interface{}
		if err := json.Unmarshal([]byte(cleanedResult), &parsedFacts); err == nil {
			for key, value := range parsedFacts {
				if strValue, ok := value.(string); ok {
					facts[key] = strValue
				} else {
					// 如果不是字符串，转换为字符串
					facts[key] = fmt.Sprintf("%v", value)
				}
			}
		} else {
			log.Debugf("Failed to parse extracted facts JSON: %v, raw: %s", err, result)
		}
	}

	return facts, nil
}

// cleanJSONResponse 清理响应内容，移除 markdown 代码块标记
func cleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	// 移除开头的 ```json 或 ```
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
	}

	// 移除结尾的 ```
	response = strings.TrimSuffix(response, "```")

	// 再次去除首尾空白字符
	return strings.TrimSpace(response)
}

// 构建对话文本
func (s *Summarizer) buildConversationText(messages []openai.ChatCompletionMessage) string {
	var builder strings.Builder
	for i, msg := range messages {
		if i > 0 {
			builder.WriteString("\n")
		}
		var role string
		switch msg.Role {
		case openai.ChatMessageRoleAssistant:
			role = "助手"
		case openai.ChatMessageRoleSystem:
			role = "系统"
		default:
			role = "用户"
		}
		builder.WriteString(fmt.Sprintf("%s: %s", role, msg.Content))
	}
	return builder.String()
}
