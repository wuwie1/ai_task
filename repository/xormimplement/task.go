package xormimplement

import (
	"ai_task/entity"
	"ai_task/model"
	"ai_task/repository"
	"fmt"
	"time"

	"xorm.io/builder"
)

// ========== TaskRepository 实现 ==========

type TaskRepository struct {
	session *Session
}

func NewTaskRepository(session *Session) repository.TaskRepository {
	return &TaskRepository{session: session}
}

func (r *TaskRepository) Upsert(req *model.UpsertTaskCondition) error {
	if req == nil {
		return fmt.Errorf("upsert request cannot be nil")
	}
	if req.ID == "" {
		return fmt.Errorf("task id is required")
	}

	// 先尝试获取现有记录
	existing := &entity.Task{}
	has, err := r.session.Table(entity.TableNameTask).
		Where(builder.Eq{entity.TaskFieldID: req.ID}).
		Get(existing)
	if err != nil {
		return fmt.Errorf("failed to check existing task: %w", err)
	}

	if has {
		// 更新现有记录
		updateData := make(map[string]interface{})
		updateData[entity.TaskFieldUpdatedAt] = time.Now()

		if req.Goal != "" {
			updateData[entity.TaskFieldGoal] = req.Goal
		}
		if req.CurrentPhase != nil {
			updateData[entity.TaskFieldCurrentPhase] = *req.CurrentPhase
		}
		if req.PhasesJSON != nil {
			updateData[entity.TaskFieldPhasesJSON] = *req.PhasesJSON
		}
		if req.QuestionsJSON != nil {
			updateData[entity.TaskFieldQuestionsJSON] = *req.QuestionsJSON
		}
		if req.DecisionsJSON != nil {
			updateData[entity.TaskFieldDecisionsJSON] = *req.DecisionsJSON
		}
		if req.ErrorsJSON != nil {
			updateData[entity.TaskFieldErrorsJSON] = *req.ErrorsJSON
		}
		if req.Status != nil {
			updateData[entity.TaskFieldStatus] = *req.Status
		}
		if req.ToolCallCount != nil {
			updateData[entity.TaskFieldToolCallCount] = *req.ToolCallCount
		}
		if req.NeedsReread != nil {
			updateData[entity.TaskFieldNeedsReread] = *req.NeedsReread
		}
		if req.CompletedAt != nil {
			updateData[entity.TaskFieldCompletedAt] = *req.CompletedAt
		}

		_, err = r.session.Table(entity.TableNameTask).
			Where(builder.Eq{entity.TaskFieldID: req.ID}).
			Update(updateData)
		if err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}
	} else {
		// 插入新记录
		status := "pending"
		if req.Status != nil {
			status = *req.Status
		}

		newTask := &entity.Task{
			ID:            req.ID,
			UserID:        req.UserID,
			SessionID:     req.SessionID,
			Goal:          req.Goal,
			Status:        status,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			ToolCallCount: 0,
			NeedsReread:   false,
		}

		if req.CurrentPhase != nil {
			newTask.CurrentPhase = *req.CurrentPhase
		}
		if req.PhasesJSON != nil {
			newTask.PhasesJSON = *req.PhasesJSON
		}
		if req.QuestionsJSON != nil {
			newTask.QuestionsJSON = *req.QuestionsJSON
		}
		if req.DecisionsJSON != nil {
			newTask.DecisionsJSON = *req.DecisionsJSON
		}
		if req.ErrorsJSON != nil {
			newTask.ErrorsJSON = *req.ErrorsJSON
		}
		if req.ToolCallCount != nil {
			newTask.ToolCallCount = *req.ToolCallCount
		}
		if req.NeedsReread != nil {
			newTask.NeedsReread = *req.NeedsReread
		}
		if req.CompletedAt != nil {
			newTask.CompletedAt = req.CompletedAt
		}

		_, err = r.session.Table(entity.TableNameTask).Insert(newTask)
		if err != nil {
			return fmt.Errorf("failed to insert task: %w", err)
		}
	}

	return nil
}

func (r *TaskRepository) Get(taskID string) (*entity.Task, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	result := &entity.Task{}
	ok, err := r.session.Table(entity.TableNameTask).
		Where(builder.Eq{entity.TaskFieldID: taskID}).
		Get(result)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if !ok {
		return nil, nil
	}

	return result, nil
}

func (r *TaskRepository) List(condition *model.TaskListCondition) ([]*entity.Task, error) {
	session := r.session.Table(entity.TableNameTask)
	var conds []builder.Cond

	if condition != nil {
		if condition.UserID != nil && *condition.UserID != "" {
			conds = append(conds, builder.Eq{entity.TaskFieldUserID: *condition.UserID})
		}
		if condition.SessionID != nil && *condition.SessionID != "" {
			conds = append(conds, builder.Eq{entity.TaskFieldSessionID: *condition.SessionID})
		}
	}

	if len(conds) > 0 {
		session = session.Where(builder.And(conds...))
	}

	var results []*entity.Task
	err := session.Desc(entity.TaskFieldCreatedAt).Find(&results)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return results, nil
}

func (r *TaskRepository) Delete(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task_id is required")
	}

	// 开启事务
	if err := r.session.Begin(); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 删除任务
	_, err := r.session.Table(entity.TableNameTask).
		Where(builder.Eq{entity.TaskFieldID: taskID}).
		Delete(&entity.Task{})
	if err != nil {
		_ = r.session.Rollback()
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// 删除关联的 findings
	_, err = r.session.Table(entity.TableNameTaskFindings).
		Where(builder.Eq{entity.TaskFindingsFieldTaskID: taskID}).
		Delete(&entity.TaskFindings{})
	if err != nil {
		_ = r.session.Rollback()
		return fmt.Errorf("failed to delete task findings: %w", err)
	}

	// 删除关联的 progress
	_, err = r.session.Table(entity.TableNameTaskProgress).
		Where(builder.Eq{entity.TaskProgressFieldTaskID: taskID}).
		Delete(&entity.TaskProgress{})
	if err != nil {
		_ = r.session.Rollback()
		return fmt.Errorf("failed to delete task progress: %w", err)
	}

	if err := r.session.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *TaskRepository) Query(condition *model.TaskQueryCondition) ([]*entity.Task, int64, error) {
	if condition == nil {
		condition = &model.TaskQueryCondition{}
	}

	// 构建查询条件
	var conds []builder.Cond
	if condition.UserID != nil && *condition.UserID != "" {
		conds = append(conds, builder.Eq{entity.TaskFieldUserID: *condition.UserID})
	}
	if condition.SessionID != nil && *condition.SessionID != "" {
		conds = append(conds, builder.Eq{entity.TaskFieldSessionID: *condition.SessionID})
	}
	if condition.Status != nil && *condition.Status != "" {
		conds = append(conds, builder.Eq{entity.TaskFieldStatus: *condition.Status})
	}
	if condition.Keyword != nil && *condition.Keyword != "" {
		conds = append(conds, builder.Like{entity.TaskFieldGoal, *condition.Keyword})
	}
	if condition.StartDate != nil {
		conds = append(conds, builder.Gte{entity.TaskFieldCreatedAt: *condition.StartDate})
	}
	if condition.EndDate != nil {
		conds = append(conds, builder.Lte{entity.TaskFieldCreatedAt: *condition.EndDate})
	}

	whereCond := builder.NewCond()
	if len(conds) > 0 {
		whereCond = builder.And(conds...)
	}

	// 计算总数
	total, err := r.session.Table(entity.TableNameTask).Where(whereCond).Count(&entity.Task{})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// 查询数据
	session := r.session.Table(entity.TableNameTask).Where(whereCond)

	// 分页
	if condition.Limit > 0 {
		session = session.Limit(condition.Limit, condition.Offset)
	}

	// 排序
	if condition.OrderBy != "" {
		if condition.OrderDesc {
			session = session.Desc(condition.OrderBy)
		} else {
			session = session.Asc(condition.OrderBy)
		}
	} else {
		session = session.Desc(entity.TaskFieldCreatedAt)
	}

	var results []*entity.Task
	err = session.Find(&results)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}

	return results, total, nil
}

func (r *TaskRepository) GetStats(userID string) (*model.TaskStats, error) {
	stats := &model.TaskStats{}

	baseSession := r.session.Table(entity.TableNameTask)
	if userID != "" {
		baseSession = baseSession.Where(builder.Eq{entity.TaskFieldUserID: userID})
	}

	// 总数
	total, err := baseSession.Count(&entity.Task{})
	if err != nil {
		return nil, fmt.Errorf("failed to count total tasks: %w", err)
	}
	stats.Total = int(total)

	// 各状态数量
	statuses := []string{"pending", "in_progress", "completed", "failed"}
	for _, status := range statuses {
		statusSession := r.session.Table(entity.TableNameTask).
			Where(builder.Eq{entity.TaskFieldStatus: status})
		if userID != "" {
			statusSession = statusSession.Where(builder.Eq{entity.TaskFieldUserID: userID})
		}
		count, err := statusSession.Count(&entity.Task{})
		if err != nil {
			return nil, fmt.Errorf("failed to count %s tasks: %w", status, err)
		}
		switch status {
		case "pending":
			stats.Pending = int(count)
		case "in_progress":
			stats.InProgress = int(count)
		case "completed":
			stats.Completed = int(count)
		case "failed":
			stats.Failed = int(count)
		}
	}

	return stats, nil
}

// ========== TaskFindingsRepository 实现 ==========

type TaskFindingsRepository struct {
	session *Session
}

func NewTaskFindingsRepository(session *Session) repository.TaskFindingsRepository {
	return &TaskFindingsRepository{session: session}
}

func (r *TaskFindingsRepository) Upsert(req *model.UpsertTaskFindingsCondition) error {
	if req == nil {
		return fmt.Errorf("upsert request cannot be nil")
	}
	if req.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}

	// 先尝试获取现有记录
	existing := &entity.TaskFindings{}
	has, err := r.session.Table(entity.TableNameTaskFindings).
		Where(builder.Eq{entity.TaskFindingsFieldTaskID: req.TaskID}).
		Get(existing)
	if err != nil {
		return fmt.Errorf("failed to check existing findings: %w", err)
	}

	if has {
		// 更新现有记录
		updateData := make(map[string]interface{})
		updateData[entity.TaskFindingsFieldUpdatedAt] = time.Now()

		if req.RequirementsJSON != nil {
			updateData[entity.TaskFindingsFieldRequirementsJSON] = *req.RequirementsJSON
		}
		if req.FindingsJSON != nil {
			updateData[entity.TaskFindingsFieldFindingsJSON] = *req.FindingsJSON
		}
		if req.ResourcesJSON != nil {
			updateData[entity.TaskFindingsFieldResourcesJSON] = *req.ResourcesJSON
		}

		_, err = r.session.Table(entity.TableNameTaskFindings).
			Where(builder.Eq{entity.TaskFindingsFieldTaskID: req.TaskID}).
			Update(updateData)
		if err != nil {
			return fmt.Errorf("failed to update findings: %w", err)
		}
	} else {
		// 插入新记录
		newFindings := &entity.TaskFindings{
			TaskID:    req.TaskID,
			UpdatedAt: time.Now(),
		}

		if req.RequirementsJSON != nil {
			newFindings.RequirementsJSON = *req.RequirementsJSON
		}
		if req.FindingsJSON != nil {
			newFindings.FindingsJSON = *req.FindingsJSON
		}
		if req.ResourcesJSON != nil {
			newFindings.ResourcesJSON = *req.ResourcesJSON
		}

		_, err = r.session.Table(entity.TableNameTaskFindings).Insert(newFindings)
		if err != nil {
			return fmt.Errorf("failed to insert findings: %w", err)
		}
	}

	return nil
}

func (r *TaskFindingsRepository) Get(taskID string) (*entity.TaskFindings, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	result := &entity.TaskFindings{}
	ok, err := r.session.Table(entity.TableNameTaskFindings).
		Where(builder.Eq{entity.TaskFindingsFieldTaskID: taskID}).
		Get(result)
	if err != nil {
		return nil, fmt.Errorf("failed to get findings: %w", err)
	}

	if !ok {
		return nil, nil
	}

	return result, nil
}

func (r *TaskFindingsRepository) Delete(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task_id is required")
	}

	_, err := r.session.Table(entity.TableNameTaskFindings).
		Where(builder.Eq{entity.TaskFindingsFieldTaskID: taskID}).
		Delete(&entity.TaskFindings{})
	if err != nil {
		return fmt.Errorf("failed to delete findings: %w", err)
	}

	return nil
}

// ========== TaskProgressRepository 实现 ==========

type TaskProgressRepository struct {
	session *Session
}

func NewTaskProgressRepository(session *Session) repository.TaskProgressRepository {
	return &TaskProgressRepository{session: session}
}

func (r *TaskProgressRepository) Upsert(req *model.UpsertTaskProgressCondition) error {
	if req == nil {
		return fmt.Errorf("upsert request cannot be nil")
	}
	if req.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}

	// 先尝试获取现有记录
	existing := &entity.TaskProgress{}
	has, err := r.session.Table(entity.TableNameTaskProgress).
		Where(builder.Eq{entity.TaskProgressFieldTaskID: req.TaskID}).
		Get(existing)
	if err != nil {
		return fmt.Errorf("failed to check existing progress: %w", err)
	}

	if has {
		// 更新现有记录
		updateData := make(map[string]interface{})
		updateData[entity.TaskProgressFieldUpdatedAt] = time.Now()

		if req.SessionDate != nil {
			updateData[entity.TaskProgressFieldSessionDate] = *req.SessionDate
		}
		if req.EntriesJSON != nil {
			updateData[entity.TaskProgressFieldEntriesJSON] = *req.EntriesJSON
		}
		if req.TestResultsJSON != nil {
			updateData[entity.TaskProgressFieldTestResultsJSON] = *req.TestResultsJSON
		}
		if req.ErrorLogJSON != nil {
			updateData[entity.TaskProgressFieldErrorLogJSON] = *req.ErrorLogJSON
		}

		_, err = r.session.Table(entity.TableNameTaskProgress).
			Where(builder.Eq{entity.TaskProgressFieldTaskID: req.TaskID}).
			Update(updateData)
		if err != nil {
			return fmt.Errorf("failed to update progress: %w", err)
		}
	} else {
		// 插入新记录
		newProgress := &entity.TaskProgress{
			TaskID:    req.TaskID,
			UpdatedAt: time.Now(),
		}

		if req.SessionDate != nil {
			newProgress.SessionDate = *req.SessionDate
		}
		if req.EntriesJSON != nil {
			newProgress.EntriesJSON = *req.EntriesJSON
		}
		if req.TestResultsJSON != nil {
			newProgress.TestResultsJSON = *req.TestResultsJSON
		}
		if req.ErrorLogJSON != nil {
			newProgress.ErrorLogJSON = *req.ErrorLogJSON
		}

		_, err = r.session.Table(entity.TableNameTaskProgress).Insert(newProgress)
		if err != nil {
			return fmt.Errorf("failed to insert progress: %w", err)
		}
	}

	return nil
}

func (r *TaskProgressRepository) Get(taskID string) (*entity.TaskProgress, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	result := &entity.TaskProgress{}
	ok, err := r.session.Table(entity.TableNameTaskProgress).
		Where(builder.Eq{entity.TaskProgressFieldTaskID: taskID}).
		Get(result)
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	if !ok {
		return nil, nil
	}

	return result, nil
}

func (r *TaskProgressRepository) Delete(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task_id is required")
	}

	_, err := r.session.Table(entity.TableNameTaskProgress).
		Where(builder.Eq{entity.TaskProgressFieldTaskID: taskID}).
		Delete(&entity.TaskProgress{})
	if err != nil {
		return fmt.Errorf("failed to delete progress: %w", err)
	}

	return nil
}
