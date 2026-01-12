package chat

import (
	"ai_task/model"
	"ai_task/pkg/clients/llm_model"
	"ai_task/repository/factory"
	"context"
	"sync"
)

var (
	serviceOnce sync.Once
	instance    *Service
)

type Service struct {
	repositoryFactory factory.Factory
	llmClient         *llm_model.ClientChatModel
}

func NewService(repositoryFactory factory.Factory) *Service {
	serviceOnce.Do(func() {

		instance = &Service{
			repositoryFactory: repositoryFactory,
			llmClient:         llm_model.GetInstance(),
		}
	})

	return instance
}

// Chat 处理聊天请求
func (s *Service) Chat(ctx context.Context, req *model.ChatRequest, options *model.MemoryContextOptionsRequest) (*model.ChatResponse, *model.Error) {
	return nil, nil
}
