package xormimplement

import (
	"ai_web/test/config"
	"ai_web/test/repository"
	"ai_web/test/repository/factory"
	"ai_web/test/repository/interfaces"
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"xorm.io/xorm"

	_ "github.com/lib/pq"
)

var once sync.Once
var instance *Factory

type Factory struct {
	// 连接 pg
	engine *xorm.Engine
	// 连接 es
}

// 获取一个factory实例
func GetRepositoryFactoryInstance() factory.Factory {
	once.Do(func() {
		instance = &Factory{
			engine: openDB(
				config.GetInstance().GetString(config.BaseDbXormType),
				config.GetInstance().GetString(config.BaseDbXormHost),
				config.GetInstance().GetString(config.BaseDbXormPort),
				config.GetInstance().GetString(config.BaseDbXormUsername),
				config.GetInstance().GetString(config.BaseDbXormName),
				config.GetInstance().GetString(config.BaseDbXormPassword),
				config.GetInstance().GetBool(config.BaseDbXormShowsql),
			),
		}
	})
	return instance
}

// 设置xorm的连接参数
func openDB(dbType string, host string, port string, userName string, name string, password string, showSql bool) *xorm.Engine {
	//拼接数据库参数
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		host,
		userName,
		password,
		name,
		port)
	//设置连接参数
	engine, err := xorm.NewEngine(dbType, dsn)
	if err != nil {
		logrus.Errorf("Database connection failed err: %v. Database name: %s", err, name)
		panic(err)
	}
	//是否展示sql文件
	engine.ShowSQL(showSql)
	return engine
}

// 创建一个会话
func (f *Factory) NewSession(ctx context.Context) interfaces.Session {
	return &Session{Session: f.engine.NewSession().Context(ctx)}
}

// NewChatMemoryChunksRepository 创建聊天记忆仓库
func (f *Factory) NewChatMemoryChunksRepository(session interfaces.Session) (repository.ChatMemoryChunksRepository, error) {
	if s, ok := session.(*Session); ok {
		return NewChatMemoryChunksRepository(s), nil
	}
	return nil, fmt.Errorf("xorm session 结构解析失败")
}

// NewUserProfileRepository 创建用户画像仓库
func (f *Factory) NewUserProfileRepository(session interfaces.Session) (repository.UserProfileRepository, error) {
	if s, ok := session.(*Session); ok {
		return NewUserProfileRepository(s), nil
	}
	return nil, fmt.Errorf("xorm session 结构解析失败")
}
