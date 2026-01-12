package repository

import (
	"ai_web/test/entity"
	"ai_web/test/model"
)

type ChatMemoryChunksRepository interface {
	Insert(data []*entity.ChatMemoryChunks) error
	Delete(id int64) error
	Update(id int64, req *model.UpdateChatMemoryChunksCondition) error
	Get(id int64) (*entity.ChatMemoryChunks, error)
	List(condition *model.GetChatMemoryChunksCondition) ([]*entity.ChatMemoryChunks, int64, error)
	ListNoCount(condition *model.GetChatMemoryChunksConditionNoCount) ([]*entity.ChatMemoryChunks, error)
	ListCount(condition *model.GetChatMemoryChunksConditionCount) (int64, error)
	// VectorSearch 向量相似度检索
	VectorSearch(condition *model.VectorSearchCondition) ([]*entity.ChatMemoryChunks, error)
	// GetRecentBySession 获取会话最近的 N 条记录
	GetRecentBySession(userID, sessionID string, limit int) ([]*entity.ChatMemoryChunks, error)
}

