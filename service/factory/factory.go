package factory

import (
	"ai_web/test/repository/factory"
	"ai_web/test/repository/xormimplement"
	"ai_web/test/service/chat"
	"ai_web/test/service/memory"
	"sync"
)

var instance *Factory
var once sync.Once

// 创建
type Factory struct {
	repositoryFactory factory.Factory
}

// 实例化instance
func init() {
	once.Do(func() {
		instance = &Factory{repositoryFactory: xormimplement.GetRepositoryFactoryInstance()}
	})
}

// 单例模式，
func GetServiceFactory() *Factory {
	return instance
}

// NewChatService 获取聊天服务
func (f *Factory) NewChatService() *chat.Service {
	return chat.NewService(f.repositoryFactory)
}

// NewMemoryService 获取记忆服务
func (f *Factory) NewMemoryService() *memory.Service {
	svc, err := memory.NewService(f.repositoryFactory)
	if err != nil {
		panic("failed to create memory service: " + err.Error())
	}
	return svc
}
