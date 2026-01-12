package factory

import (
	"ai_task/repository"
	"ai_task/repository/interfaces"
	"context"
)

type Factory interface {
	NewSession(ctx context.Context) interfaces.Session
	NewUserProfileRepository(session interfaces.Session) (repository.UserProfileRepository, error)
	NewTaskRepository(session interfaces.Session) (repository.TaskRepository, error)
	NewTaskFindingsRepository(session interfaces.Session) (repository.TaskFindingsRepository, error)
	NewTaskProgressRepository(session interfaces.Session) (repository.TaskProgressRepository, error)
}
