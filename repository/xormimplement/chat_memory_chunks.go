package xormimplement

import (
	"ai_web/test/entity"
	"ai_web/test/model"
	"ai_web/test/repository"
	"fmt"

	"xorm.io/builder"
)

type ChatMemoryChunksRepository struct {
	session *Session
}

func NewChatMemoryChunksRepository(session *Session) repository.ChatMemoryChunksRepository {
	return &ChatMemoryChunksRepository{session: session}
}

func buildChatMemoryChunksQueryConditions(condition *model.GetChatMemoryChunksCondition) builder.Cond {
	var conds []builder.Cond

	if condition.UserID != nil && *condition.UserID != "" {
		conds = append(conds, builder.Eq{entity.ChatMemoryChunksFieldUserID: *condition.UserID})
	}
	if condition.SessionID != nil && *condition.SessionID != "" {
		conds = append(conds, builder.Eq{entity.ChatMemoryChunksFieldSessionID: *condition.SessionID})
	}
	if condition.StartTS != nil {
		conds = append(conds, builder.Gte{entity.ChatMemoryChunksFieldStartTS: *condition.StartTS})
	}
	if condition.EndTS != nil {
		conds = append(conds, builder.Lte{entity.ChatMemoryChunksFieldEndTS: *condition.EndTS})
	}

	if len(conds) == 0 {
		return nil
	}
	return builder.And(conds...)
}

func buildChatMemoryChunksQueryConditionsNoCount(condition *model.GetChatMemoryChunksConditionNoCount) builder.Cond {
	var conds []builder.Cond

	if condition.UserID != nil && *condition.UserID != "" {
		conds = append(conds, builder.Eq{entity.ChatMemoryChunksFieldUserID: *condition.UserID})
	}
	if condition.SessionID != nil && *condition.SessionID != "" {
		conds = append(conds, builder.Eq{entity.ChatMemoryChunksFieldSessionID: *condition.SessionID})
	}
	if condition.StartTS != nil {
		conds = append(conds, builder.Gte{entity.ChatMemoryChunksFieldStartTS: *condition.StartTS})
	}
	if condition.EndTS != nil {
		conds = append(conds, builder.Lte{entity.ChatMemoryChunksFieldEndTS: *condition.EndTS})
	}

	if len(conds) == 0 {
		return nil
	}
	return builder.And(conds...)
}

func buildChatMemoryChunksQueryConditionsCount(condition *model.GetChatMemoryChunksConditionCount) builder.Cond {
	var conds []builder.Cond

	if condition.UserID != nil && *condition.UserID != "" {
		conds = append(conds, builder.Eq{entity.ChatMemoryChunksFieldUserID: *condition.UserID})
	}
	if condition.SessionID != nil && *condition.SessionID != "" {
		conds = append(conds, builder.Eq{entity.ChatMemoryChunksFieldSessionID: *condition.SessionID})
	}

	if len(conds) == 0 {
		return nil
	}
	return builder.And(conds...)
}

func (r *ChatMemoryChunksRepository) Insert(data []*entity.ChatMemoryChunks) error {
	if len(data) == 0 {
		return fmt.Errorf("chat_memory_chunks data cannot be empty")
	}

	for _, item := range data {
		if item == nil {
			return fmt.Errorf("chat_memory_chunks item cannot be nil")
		}
	}

	_, err := r.session.Table(entity.TableNameChatMemoryChunks).Insert(data)
	if err != nil {
		return fmt.Errorf("failed to insert chat_memory_chunks: %w", err)
	}

	return nil
}

func (r *ChatMemoryChunksRepository) Delete(id int64) error {
	if id <= 0 {
		return fmt.Errorf("chat_memory_chunks id must be greater than 0")
	}

	_, err := r.session.Table(entity.TableNameChatMemoryChunks).
		Where(builder.Eq{entity.ChatMemoryChunksFieldID: id}).
		Delete(&entity.ChatMemoryChunks{})
	if err != nil {
		return fmt.Errorf("failed to delete chat_memory_chunks: %w", err)
	}

	return nil
}

func (r *ChatMemoryChunksRepository) Update(id int64, req *model.UpdateChatMemoryChunksCondition) error {
	if id <= 0 {
		return fmt.Errorf("chat_memory_chunks id must be greater than 0")
	}
	if req == nil {
		return fmt.Errorf("update request cannot be nil")
	}

	updateData := make(map[string]interface{})
	if req.Text != nil {
		updateData[entity.ChatMemoryChunksFieldText] = *req.Text
	}
	if req.Summary != nil {
		updateData[entity.ChatMemoryChunksFieldSummary] = *req.Summary
	}
	if req.Embedding != nil {
		updateData[entity.ChatMemoryChunksFieldEmbedding] = *req.Embedding
	}
	if req.Meta != nil {
		updateData[entity.ChatMemoryChunksFieldMeta] = *req.Meta
	}

	if len(updateData) == 0 {
		return fmt.Errorf("at least one field must be updated")
	}

	_, err := r.session.Table(entity.TableNameChatMemoryChunks).
		Where(builder.Eq{entity.ChatMemoryChunksFieldID: id}).
		Update(updateData)
	if err != nil {
		return fmt.Errorf("failed to update chat_memory_chunks: %w", err)
	}

	return nil
}

func (r *ChatMemoryChunksRepository) Get(id int64) (*entity.ChatMemoryChunks, error) {
	if id <= 0 {
		return nil, fmt.Errorf("chat_memory_chunks id must be greater than 0")
	}

	result := &entity.ChatMemoryChunks{}
	ok, err := r.session.Table(entity.TableNameChatMemoryChunks).
		Where(builder.Eq{entity.ChatMemoryChunksFieldID: id}).
		Get(result)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat_memory_chunks: %w", err)
	}

	if !ok {
		return nil, nil
	}

	return result, nil
}

func (r *ChatMemoryChunksRepository) List(condition *model.GetChatMemoryChunksCondition) ([]*entity.ChatMemoryChunks, int64, error) {
	if condition == nil {
		return nil, 0, fmt.Errorf("get condition cannot be nil")
	}

	cond := buildChatMemoryChunksQueryConditions(condition)

	session := r.session.Table(entity.TableNameChatMemoryChunks)
	if cond != nil {
		session = session.Where(cond)
	}

	pagerOrder(session, condition, WithDefaultOrderField(entity.ChatMemoryChunksFieldStartTS))

	var results []*entity.ChatMemoryChunks
	total, err := session.FindAndCount(&results)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list chat_memory_chunks: %w", err)
	}

	return results, total, nil
}

func (r *ChatMemoryChunksRepository) ListNoCount(condition *model.GetChatMemoryChunksConditionNoCount) ([]*entity.ChatMemoryChunks, error) {
	if condition == nil {
		return nil, fmt.Errorf("get condition cannot be nil")
	}

	cond := buildChatMemoryChunksQueryConditionsNoCount(condition)

	session := r.session.Table(entity.TableNameChatMemoryChunks)
	if cond != nil {
		session = session.Where(cond)
	}

	var results []*entity.ChatMemoryChunks
	if err := session.Find(&results); err != nil {
		return nil, fmt.Errorf("failed to list chat_memory_chunks: %w", err)
	}

	return results, nil
}

func (r *ChatMemoryChunksRepository) ListCount(condition *model.GetChatMemoryChunksConditionCount) (int64, error) {
	if condition == nil {
		return 0, fmt.Errorf("get condition cannot be nil")
	}

	cond := buildChatMemoryChunksQueryConditionsCount(condition)

	session := r.session.Table(entity.TableNameChatMemoryChunks)
	if cond != nil {
		session = session.Where(cond)
	}

	total, err := session.Count(&entity.ChatMemoryChunks{})
	if err != nil {
		return 0, fmt.Errorf("failed to count chat_memory_chunks: %w", err)
	}

	return total, nil
}

// VectorSearch 向量相似度检索（使用 pgvector 的余弦相似度）
func (r *ChatMemoryChunksRepository) VectorSearch(condition *model.VectorSearchCondition) ([]*entity.ChatMemoryChunks, error) {
	if condition == nil {
		return nil, fmt.Errorf("vector search condition cannot be nil")
	}
	if condition.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if condition.QueryVector == "" {
		return nil, fmt.Errorf("query_vector is required")
	}
	if condition.Limit <= 0 {
		condition.Limit = 10 // 默认返回10条
	}

	// 构建 SQL 查询
	// 使用 pgvector 的 <=> 操作符进行余弦相似度计算
	// 1 - (embedding <=> query_vector) 得到相似度分数（越大越相似）
	sql := fmt.Sprintf(`
		SELECT id, user_id, session_id, start_ts, end_ts, text, summary, embedding, meta,
		       1 - (embedding <=> '%s'::vector) as similarity
		FROM %s
		WHERE user_id = $1
	`, condition.QueryVector, entity.TableNameChatMemoryChunks)

	args := []interface{}{condition.UserID}
	argIndex := 2

	// 添加可选的过滤条件
	if condition.SessionID != nil && *condition.SessionID != "" {
		sql += fmt.Sprintf(" AND session_id = $%d", argIndex)
		args = append(args, *condition.SessionID)
		argIndex++
	}
	if condition.StartTS != nil {
		sql += fmt.Sprintf(" AND start_ts >= $%d", argIndex)
		args = append(args, *condition.StartTS)
		argIndex++
	}
	if condition.EndTS != nil {
		sql += fmt.Sprintf(" AND end_ts <= $%d", argIndex)
		args = append(args, *condition.EndTS)
		argIndex++
	}
	if condition.Threshold != nil {
		sql += fmt.Sprintf(" AND (1 - (embedding <=> '%s'::vector)) >= $%d", condition.QueryVector, argIndex)
		args = append(args, *condition.Threshold)
	}

	// 按相似度降序排序并限制数量
	sql += fmt.Sprintf(" ORDER BY similarity DESC LIMIT %d", condition.Limit)

	var results []*entity.ChatMemoryChunks
	err := r.session.SQL(sql, args...).Find(&results)
	if err != nil {
		return nil, fmt.Errorf("failed to vector search chat_memory_chunks: %w", err)
	}

	return results, nil
}

// GetRecentBySession 获取会话最近的 N 条记录
func (r *ChatMemoryChunksRepository) GetRecentBySession(userID, sessionID string, limit int) ([]*entity.ChatMemoryChunks, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if limit <= 0 {
		limit = 10
	}

	var results []*entity.ChatMemoryChunks
	err := r.session.Table(entity.TableNameChatMemoryChunks).
		Where(builder.Eq{
			entity.ChatMemoryChunksFieldUserID:    userID,
			entity.ChatMemoryChunksFieldSessionID: sessionID,
		}).
		OrderBy(entity.ChatMemoryChunksFieldStartTS + " DESC").
		Limit(limit).
		Find(&results)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent chat_memory_chunks: %w", err)
	}

	// 反转结果，使其按时间升序排列
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return results, nil
}
