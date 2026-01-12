package xormimplement

import (
	"github.com/pkg/errors"
	"xorm.io/xorm"
)

//由xorm（ORM）框架具体实现
type Session struct {
	*xorm.Session
}

//实现session接口，begin：开启一个会话
func (s *Session) Begin() error {
	if err := s.Session.Begin(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

//实现session接口，close：关闭一个会话
func (s *Session) Close() error {
	if err := s.Session.Close(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

//实现session接口，commit：提交sql
func (s *Session) Commit() error {
	if err := s.Session.Commit(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

//实现session接口，Rollback：执行操作回滚
func (s *Session) Rollback() error {
	if err := s.Session.Rollback(); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
