package repository

import (
	"ai_task/entity"
	"ai_task/model"
)

// TaskRepository 任务仓库接口
type TaskRepository interface {
	// ========== 任务 CRUD ==========

	// Upsert 创建或更新任务
	Upsert(req *model.UpsertTaskCondition) error
	// Get 获取单个任务
	Get(taskID string) (*entity.Task, error)
	// List 列出任务
	List(condition *model.TaskListCondition) ([]*entity.Task, error)
	// Delete 删除任务（同时删除相关的 findings 和 progress）
	Delete(taskID string) error
	// Query 高级查询（支持分页、排序、过滤）
	Query(condition *model.TaskQueryCondition) ([]*entity.Task, int64, error)
	// GetStats 获取任务统计
	GetStats(userID string) (*model.TaskStats, error)
}

// TaskFindingsRepository 任务发现仓库接口
type TaskFindingsRepository interface {
	// Upsert 创建或更新任务发现
	Upsert(req *model.UpsertTaskFindingsCondition) error
	// Get 获取任务发现
	Get(taskID string) (*entity.TaskFindings, error)
	// Delete 删除任务发现
	Delete(taskID string) error
}

// TaskProgressRepository 任务进度仓库接口
type TaskProgressRepository interface {
	// Upsert 创建或更新任务进度
	Upsert(req *model.UpsertTaskProgressCondition) error
	// Get 获取任务进度
	Get(taskID string) (*entity.TaskProgress, error)
	// Delete 删除任务进度
	Delete(taskID string) error
}
