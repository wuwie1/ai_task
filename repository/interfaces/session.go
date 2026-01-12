package interfaces

//定义数据库连接会话接口
type Session interface {
	Begin() error
	Close() error
	Commit() error
	Rollback() error
}
