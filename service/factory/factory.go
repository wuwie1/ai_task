package factory

import (
	"ai_task/repository/factory"
	"ai_task/repository/xormimplement"
	"ai_task/service/chat"
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
