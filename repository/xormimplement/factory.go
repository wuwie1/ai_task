package xormimplement

import (
	"ai_task/config"
	"ai_task/repository"
	"ai_task/repository/factory"
	"ai_task/repository/interfaces"
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

// NewUserProfileRepository 创建用户画像仓库
func (f *Factory) NewUserProfileRepository(session interfaces.Session) (repository.UserProfileRepository, error) {
	if s, ok := session.(*Session); ok {
		return NewUserProfileRepository(s), nil
	}
	return nil, fmt.Errorf("xorm session 结构解析失败")
}

// NewTaskRepository 创建任务仓库
func (f *Factory) NewTaskRepository(session interfaces.Session) (repository.TaskRepository, error) {
	if s, ok := session.(*Session); ok {
		return NewTaskRepository(s), nil
	}
	return nil, fmt.Errorf("xorm session 结构解析失败")
}

// NewTaskFindingsRepository 创建任务发现仓库
func (f *Factory) NewTaskFindingsRepository(session interfaces.Session) (repository.TaskFindingsRepository, error) {
	if s, ok := session.(*Session); ok {
		return NewTaskFindingsRepository(s), nil
	}
	return nil, fmt.Errorf("xorm session 结构解析失败")
}

// NewTaskProgressRepository 创建任务进度仓库
func (f *Factory) NewTaskProgressRepository(session interfaces.Session) (repository.TaskProgressRepository, error) {
	if s, ok := session.(*Session); ok {
		return NewTaskProgressRepository(s), nil
	}
	return nil, fmt.Errorf("xorm session 结构解析失败")
}
