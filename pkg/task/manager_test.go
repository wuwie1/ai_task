package task

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	// 使用临时目录
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{
		StoragePath:     tmpDir,
		RereadThreshold: 10,
		MaxRetries:      3,
	}

	manager, err := NewManager(config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestCreateTask(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{
		StoragePath:     tmpDir,
		RereadThreshold: 10,
	}

	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "实现一个任务管理系统",
	}

	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, req.UserID, task.UserID)
	assert.Equal(t, req.SessionID, task.SessionID)
	assert.Equal(t, req.Goal, task.Goal)
	assert.Equal(t, TaskStatusPending, task.Status)
	assert.Len(t, task.Phases, 5) // 默认5个阶段
}

func TestGetTask(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	// 创建任务
	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	created, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 获取任务
	task, err := manager.GetTask(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, created.ID, task.ID)
	assert.Equal(t, created.Goal, task.Goal)
}

func TestGetTaskContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	created, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 获取完整上下文
	taskCtx, err := manager.GetTaskContext(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, taskCtx)
	assert.NotNil(t, taskCtx.Task)
	assert.NotNil(t, taskCtx.Findings)
	assert.NotNil(t, taskCtx.Progress)
}

func TestUpdatePhaseStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 更新阶段状态
	err = manager.UpdatePhaseStatus(ctx, task.ID, "phase_1", PhaseStatusInProgress)
	require.NoError(t, err)

	// 验证更新
	updated, err := manager.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, PhaseStatusInProgress, updated.Phases[0].Status)
	assert.NotNil(t, updated.Phases[0].StartedAt)
}

func TestCompleteStep(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 完成步骤
	err = manager.CompleteStep(ctx, task.ID, "phase_1", "step_1_1", "步骤完成")
	require.NoError(t, err)

	// 验证
	updated, err := manager.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.True(t, updated.Phases[0].Steps[0].Completed)
	assert.Equal(t, "步骤完成", updated.Phases[0].Steps[0].Result)
}

func TestRecordError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 记录错误
	err = manager.RecordError(ctx, task.ID, "测试错误", 1, "已解决")
	require.NoError(t, err)

	// 验证
	updated, err := manager.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, updated.Errors, 1)
	assert.Equal(t, "测试错误", updated.Errors[0].Error)
	assert.Equal(t, 1, updated.Errors[0].Attempt)
}

func TestAddDecision(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 添加决策
	err = manager.AddDecision(ctx, task.ID, "使用 Go 语言", "Go 语言性能好")
	require.NoError(t, err)

	// 验证
	updated, err := manager.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, updated.Decisions, 1)
	assert.Equal(t, "使用 Go 语言", updated.Decisions[0].Decision)
}

func TestAddFinding(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 添加发现
	err = manager.AddFinding(ctx, task.ID, "research", "发现了一个好方法", "https://example.com")
	require.NoError(t, err)

	// 验证
	taskCtx, err := manager.GetTaskContext(ctx, task.ID)
	require.NoError(t, err)
	assert.Len(t, taskCtx.Findings.Findings, 1)
	assert.Equal(t, "research", taskCtx.Findings.Findings[0].Category)
}

func TestRecordViewAction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{
		StoragePath:          tmpDir,
		TwoActionRuleEnabled: true,
	}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 第一次视图操作
	needsSave, err := manager.RecordViewAction(ctx, task.ID, ActionTypeView)
	require.NoError(t, err)
	assert.False(t, needsSave) // 第一次不需要保存

	// 第二次视图操作
	needsSave, err = manager.RecordViewAction(ctx, task.ID, ActionTypeView)
	require.NoError(t, err)
	assert.True(t, needsSave) // 第二次需要保存（2动作规则）
}

func TestIncrementToolCallCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{
		StoragePath:     tmpDir,
		RereadThreshold: 3, // 设置较小的阈值用于测试
	}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试目标",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 多次增加计数
	for i := 0; i < 2; i++ {
		needsReread, err := manager.IncrementToolCallCount(ctx, task.ID)
		require.NoError(t, err)
		assert.False(t, needsReread)
	}

	// 第3次应该触发重读
	needsReread, err := manager.IncrementToolCallCount(ctx, task.ID)
	require.NoError(t, err)
	assert.True(t, needsReread)
}

func TestGetTaskSummary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试摘要功能",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 添加一些数据
	_ = manager.AddDecision(ctx, task.ID, "决策1", "理由1")
	_ = manager.RecordError(ctx, task.ID, "错误1", 1, "")

	// 获取摘要
	summary, err := manager.GetTaskSummary(ctx, task.ID)
	require.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, task.ID, summary.TaskID)
	assert.Equal(t, task.Goal, summary.Goal)
	assert.Contains(t, summary.KeyDecisions, "决策1")
	assert.Contains(t, summary.RecentErrors, "错误1")
}

func TestCheckCompletion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试完成检查",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 初始状态应该是未完成
	complete, incomplete, err := manager.CheckCompletion(ctx, task.ID)
	require.NoError(t, err)
	assert.False(t, complete)
	assert.Len(t, incomplete, 5) // 5个未完成阶段

	// 完成所有阶段
	for _, phase := range task.Phases {
		err = manager.UpdatePhaseStatus(ctx, task.ID, phase.ID, PhaseStatusComplete)
		require.NoError(t, err)
	}

	// 现在应该完成了
	complete, incomplete, err = manager.CheckCompletion(ctx, task.ID)
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Len(t, incomplete, 0)
}

func TestListTasks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	// 创建多个任务
	for i := 0; i < 3; i++ {
		req := &PlanRequest{
			UserID:    "user_123",
			SessionID: "session_456",
			Goal:      "测试任务",
		}
		_, err := manager.CreateTask(ctx, req)
		require.NoError(t, err)
	}

	// 列出任务
	tasks, err := manager.ListTasks(ctx, "user_123", "session_456")
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestDeleteTask(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "task_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := &TaskManagerConfig{StoragePath: tmpDir}
	manager, err := NewManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	req := &PlanRequest{
		UserID:    "user_123",
		SessionID: "session_456",
		Goal:      "测试删除",
	}
	task, err := manager.CreateTask(ctx, req)
	require.NoError(t, err)

	// 删除任务
	err = manager.DeleteTask(ctx, task.ID)
	require.NoError(t, err)

	// 验证已删除
	deleted, err := manager.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestFileStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	// 创建任务
	now := time.Now()
	task := &Task{
		ID:           "test_task_123",
		UserID:       "user_123",
		SessionID:    "session_456",
		Goal:         "测试存储",
		CurrentPhase: "phase_1",
		Status:       TaskStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 保存
	err = storage.SaveTask(task)
	require.NoError(t, err)

	// 加载
	loaded, err := storage.LoadTask(task.ID)
	require.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, task.ID, loaded.ID)
	assert.Equal(t, task.Goal, loaded.Goal)

	// 验证 Markdown 文件也被创建
	_, err = os.Stat(tmpDir + "/" + task.ID + "/task_plan.md")
	assert.NoError(t, err)
}
