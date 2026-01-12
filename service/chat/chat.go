package chat

import (
	"ai_web/test/config"
	"ai_web/test/model"
	"ai_web/test/pkg/clients/llm_model"
	"ai_web/test/repository/factory"
	"ai_web/test/service/memory"
	"context"
	"fmt"
	"sync"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

var (
	serviceOnce sync.Once
	instance    *Service
)

type Service struct {
	repositoryFactory factory.Factory
	memoryService     *memory.Service
	llmClient         *llm_model.ClientChatModel
}

func NewService(repositoryFactory factory.Factory) *Service {
	serviceOnce.Do(func() {
		memoryService, err := memory.NewService(repositoryFactory)
		if err != nil {
			panic("failed to create memory service: " + err.Error())
		}

		instance = &Service{
			repositoryFactory: repositoryFactory,
			memoryService:     memoryService,
			llmClient:         llm_model.GetInstance(),
		}
	})

	return instance
}

// buildMemoryOptions 构建记忆配置选项
// 优先级：请求参数 > 配置文件 > 代码默认值
func (s *Service) buildMemoryOptions(requestOptions *model.MemoryContextOptionsRequest) (*memory.MemoryContextOptions, *model.Error) {
	cfg := config.GetInstance()

	// 从配置文件读取默认值
	memoryOptions := &memory.MemoryContextOptions{
		EnableSessionMemory: cfg.GetBoolOrDefault(config.MemoryEnableSessionMemory, true),
		EnableChunking:      cfg.GetBoolOrDefault(config.MemoryEnableChunking, true),
		SessionMemoryLimit:  cfg.GetIntOrDefault(config.MemorySessionMemoryLimit, 10),
		SemanticMemoryLimit: cfg.GetIntOrDefault(config.MemorySemanticMemoryLimit, 5),
		SemanticThreshold:   cfg.GetFloat64OrDefault(config.MemorySemanticThreshold, 0.7),
		CompressThreshold:   cfg.GetIntOrDefault(config.MemoryCompressThreshold, 20),
		EnableSummary:       cfg.GetBoolOrDefault(config.MemoryEnableSummary, false),
		EnableAutoExtract:   cfg.GetBoolOrDefault(config.MemoryEnableAutoExtract, false),
		ChunkMaxSize:        cfg.GetIntOrDefault(config.MemoryChunkMaxSize, 1000),
		ChunkOverlap:        cfg.GetIntOrDefault(config.MemoryChunkOverlap, 100),
		ChunkMinSize:        cfg.GetIntOrDefault(config.MemoryChunkMinSize, 200),
		ChunkStrategy:       cfg.GetStringOrDefault(config.MemoryChunkStrategy, "paragraph"),
	}

	// 如果请求中提供了配置，则覆盖默认值
	if requestOptions != nil {
		s.applyRequestOptions(memoryOptions, requestOptions)
	}

	// 参数校验
	if err := s.validateMemoryOptions(memoryOptions); err != nil {
		return nil, err
	}

	return memoryOptions, nil
}

// applyRequestOptions 应用请求中的配置选项（覆盖配置文件的值）
func (s *Service) applyRequestOptions(memoryOptions *memory.MemoryContextOptions, requestOptions *model.MemoryContextOptionsRequest) {
	// 功能开关
	if requestOptions.EnableSessionMemory != nil {
		memoryOptions.EnableSessionMemory = *requestOptions.EnableSessionMemory
	}
	if requestOptions.EnableChunking != nil {
		memoryOptions.EnableChunking = *requestOptions.EnableChunking
	}
	// 记忆配置
	if requestOptions.SessionMemoryLimit != nil {
		memoryOptions.SessionMemoryLimit = *requestOptions.SessionMemoryLimit
	}
	if requestOptions.SemanticMemoryLimit != nil {
		memoryOptions.SemanticMemoryLimit = *requestOptions.SemanticMemoryLimit
	}
	if requestOptions.SemanticThreshold != nil {
		memoryOptions.SemanticThreshold = *requestOptions.SemanticThreshold
	}
	if requestOptions.CompressThreshold != nil {
		memoryOptions.CompressThreshold = *requestOptions.CompressThreshold
	}
	if requestOptions.EnableSummary != nil {
		memoryOptions.EnableSummary = *requestOptions.EnableSummary
	}
	if requestOptions.EnableAutoExtract != nil {
		memoryOptions.EnableAutoExtract = *requestOptions.EnableAutoExtract
	}
	// 分块配置
	if requestOptions.ChunkMaxSize != nil {
		memoryOptions.ChunkMaxSize = *requestOptions.ChunkMaxSize
	}
	if requestOptions.ChunkOverlap != nil {
		memoryOptions.ChunkOverlap = *requestOptions.ChunkOverlap
	}
	if requestOptions.ChunkMinSize != nil {
		memoryOptions.ChunkMinSize = *requestOptions.ChunkMinSize
	}
	if requestOptions.ChunkStrategy != nil {
		memoryOptions.ChunkStrategy = *requestOptions.ChunkStrategy
	}
}

// validateMemoryOptions 校验记忆配置选项
func (s *Service) validateMemoryOptions(options *memory.MemoryContextOptions) *model.Error {
	// 校验 SessionMemoryLimit
	if options.SessionMemoryLimit < 1 {
		return model.NewError(model.ErrorParams, fmt.Errorf("session_memory_limit must be greater than 0, got %d", options.SessionMemoryLimit))
	}
	if options.SessionMemoryLimit > 1000 {
		return model.NewError(model.ErrorParams, fmt.Errorf("session_memory_limit must be less than or equal to 1000, got %d", options.SessionMemoryLimit))
	}

	// 校验 SemanticMemoryLimit
	if options.SemanticMemoryLimit < 1 {
		return model.NewError(model.ErrorParams, fmt.Errorf("semantic_memory_limit must be greater than 0, got %d", options.SemanticMemoryLimit))
	}
	if options.SemanticMemoryLimit > 100 {
		return model.NewError(model.ErrorParams, fmt.Errorf("semantic_memory_limit must be less than or equal to 100, got %d", options.SemanticMemoryLimit))
	}

	// 校验 SemanticThreshold
	if options.SemanticThreshold < 0 || options.SemanticThreshold > 1 {
		return model.NewError(model.ErrorParams, fmt.Errorf("semantic_threshold must be between 0 and 1, got %f", options.SemanticThreshold))
	}

	// 校验 CompressThreshold
	if options.CompressThreshold < 0 {
		return model.NewError(model.ErrorParams, fmt.Errorf("compress_threshold must be greater than or equal to 0, got %d", options.CompressThreshold))
	}

	// 校验分块配置
	if options.ChunkMaxSize < 1 {
		return model.NewError(model.ErrorParams, fmt.Errorf("chunk_max_size must be greater than 0, got %d", options.ChunkMaxSize))
	}
	if options.ChunkMaxSize > 10000 {
		return model.NewError(model.ErrorParams, fmt.Errorf("chunk_max_size must be less than or equal to 10000, got %d", options.ChunkMaxSize))
	}

	if options.ChunkOverlap < 0 {
		return model.NewError(model.ErrorParams, fmt.Errorf("chunk_overlap must be greater than or equal to 0, got %d", options.ChunkOverlap))
	}
	if options.ChunkOverlap >= options.ChunkMaxSize {
		return model.NewError(model.ErrorParams, fmt.Errorf("chunk_overlap (%d) must be less than chunk_max_size (%d)", options.ChunkOverlap, options.ChunkMaxSize))
	}

	if options.ChunkMinSize < 1 {
		return model.NewError(model.ErrorParams, fmt.Errorf("chunk_min_size must be greater than 0, got %d", options.ChunkMinSize))
	}
	if options.ChunkMinSize > options.ChunkMaxSize {
		return model.NewError(model.ErrorParams, fmt.Errorf("chunk_min_size (%d) must be less than or equal to chunk_max_size (%d)", options.ChunkMinSize, options.ChunkMaxSize))
	}

	// 校验 ChunkStrategy
	validStrategies := map[string]bool{
		"paragraph": true,
		"sentence":  true,
		"fixed":     true,
	}
	if !validStrategies[options.ChunkStrategy] {
		return model.NewError(model.ErrorParams, fmt.Errorf("chunk_strategy must be one of [paragraph, sentence, fixed], got %s", options.ChunkStrategy))
	}

	return nil
}

// Chat 处理聊天请求
func (s *Service) Chat(ctx context.Context, req *model.ChatRequest, options *model.MemoryContextOptionsRequest) (*model.ChatResponse, *model.Error) {
	// 构建并校验记忆配置选项
	memoryOptions, err := s.buildMemoryOptions(options)
	if err != nil {
		return nil, err
	}

	// 构建增强的 messages
	openaiMessages, err := s.memoryService.BuildContextWithMemory(
		ctx,
		req.UserID,
		req.SessionID,
		req.Messages,
		memoryOptions,
	)
	if err != nil {
		return nil, err
	}

	var assistantMessage string
	// 流式或非流式调用
	if req.Stream {
		// 流式返回
		// 注意：流式返回时，需要在客户端收集完整响应后再保存记忆
		if err := s.llmClient.PostChatCompletions(&ctx, openaiMessages); err != nil {
			return nil, model.NewError(model.ErrorDB, fmt.Errorf("failed to stream chat completion: %w", err))
		}
		// TODO: 流式返回时，可以通过中间件或回调来收集完整响应并保存记忆
	} else {
		// 非流式返回
		response, err := s.llmClient.PostChatCompletionsNonStream(ctx, openaiMessages)
		if err != nil {
			return nil, model.NewError(model.ErrorDB, fmt.Errorf("failed to get chat completion: %w", err))
		}

		if len(response.Choices) == 0 {
			return nil, model.NewError(model.ErrorDB, fmt.Errorf("no response from LLM"))
		}

		assistantMessage = response.Choices[0].Message.Content
		// 保存对话到记忆
		conversationMessages := append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: assistantMessage,
		})

		// 保存对话到记忆（传递 memoryOptions 以支持摘要和压缩）
		if err := s.memoryService.SaveSessionMemory(
			ctx,
			req.UserID,
			req.SessionID,
			conversationMessages,
			memoryOptions,
		); err != nil {
			log.Warnf("Failed to save session memory: %v", err)
		}
	}

	return &model.ChatResponse{
		Message:   assistantMessage,
		SessionID: req.SessionID,
	}, nil
}
