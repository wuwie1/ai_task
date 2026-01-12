package factory

import (
	"ai_web/test/repository"
	"ai_web/test/repository/interfaces"
	"context"
)

type Factory interface {
	NewSession(ctx context.Context) interfaces.Session
	NewChatMemoryChunksRepository(session interfaces.Session) (repository.ChatMemoryChunksRepository, error)
	NewUserProfileRepository(session interfaces.Session) (repository.UserProfileRepository, error)
}
