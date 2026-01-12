package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Manager 任务管理器
// 实现 Manus 的核心上下文工程原则：
// 1. 文件系统作为外部记忆
// 2. 通过复述操纵注意力
// 3. 保留错误信息
// 4. 2动作规则
type Manager struct {
	config  *TaskManagerConfig
	storage Storage
	mu      sync.RWMutex

	// 运行时状态
	activeTasks     map[string]*TaskContext // 活跃任务缓存
	viewActionCount map[string]int          // 视图动作计数（用于2动作规则）
}

// NewManager 创建任务管理器
func NewManager(config *TaskManagerConfig) (*Manager, error) {
	if config == nil {
		config = DefaultTaskManagerConfig()
	}

	storage, err := NewFileStorage(config.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &Manager{
		config:          config,
		storage:         storage,
		activeTasks:     make(map[string]*TaskContext),
		viewActionCount: make(map[string]int),
	}, nil
}

// CreateTask 创建新任务
// 遵循规则1：先创建计划（Create Plan First）
func (m *Manager) CreateTask(ctx context.Context, req *PlanRequest) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskID := uuid.New().String()
	now := time.Now()

	task := &Task{
		ID:           taskID,
		UserID:       req.UserID,
		SessionID:    req.SessionID,
		Goal:         req.Goal,
		CurrentPhase: "phase_1",
		Phases:       m.createDefaultPhases(),
		KeyQuestions: []string{},
		Decisions:    []Decision{},
		Errors:       []ErrorRecord{},
		Status:       TaskStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 初始化发现和进度
	findings := &TaskFindings{
		TaskID:       taskID,
		Requirements: []string{},
		Findings:     []Finding{},
		Resources:    []string{},
		UpdatedAt:    now,
	}

	progress := &TaskProgress{
		TaskID:      taskID,
		SessionDate: now.Format("2006-01-02"),
		Entries:     []ProgressEntry{},
		TestResults: []TestResult{},
		ErrorLog:    []ErrorRecord{},
		UpdatedAt:   now,
	}

	taskCtx := &TaskContext{
		Task:     task,
		Findings: findings,
		Progress: progress,
	}

	// 保存到文件系统
	if err := m.storage.SaveContext(taskCtx); err != nil {
		return nil, fmt.Errorf("failed to save task context: %w", err)
	}

	// 缓存活跃任务
	m.activeTasks[taskID] = taskCtx

	log.Infof("Created task %s with goal: %s", taskID, req.Goal)

	return task, nil
}

// createDefaultPhases 创建默认阶段
func (m *Manager) createDefaultPhases() []TaskPhase {
	return []TaskPhase{
		{
			ID:          "phase_1",
			Name:        "Requirements & Discovery",
			Description: "理解需求并收集信息",
			Status:      PhaseStatusPending,
			Order:       1,
			Steps: []TaskStep{
				{ID: "step_1_1", Description: "理解用户意图", Completed: false},
				{ID: "step_1_2", Description: "识别约束和需求", Completed: false},
				{ID: "step_1_3", Description: "记录发现到 findings", Completed: false},
			},
		},
		{
			ID:          "phase_2",
			Name:        "Planning & Structure",
			Description: "规划方案和结构",
			Status:      PhaseStatusPending,
			Order:       2,
			Steps: []TaskStep{
				{ID: "step_2_1", Description: "定义技术方案", Completed: false},
				{ID: "step_2_2", Description: "创建项目结构", Completed: false},
				{ID: "step_2_3", Description: "记录决策和理由", Completed: false},
			},
		},
		{
			ID:          "phase_3",
			Name:        "Implementation",
			Description: "执行实现",
			Status:      PhaseStatusPending,
			Order:       3,
			Steps: []TaskStep{
				{ID: "step_3_1", Description: "按步骤执行计划", Completed: false},
				{ID: "step_3_2", Description: "先写代码再执行", Completed: false},
				{ID: "step_3_3", Description: "增量测试", Completed: false},
			},
		},
		{
			ID:          "phase_4",
			Name:        "Testing & Verification",
			Description: "测试和验证",
			Status:      PhaseStatusPending,
			Order:       4,
			Steps: []TaskStep{
				{ID: "step_4_1", Description: "验证所有需求已满足", Completed: false},
				{ID: "step_4_2", Description: "记录测试结果", Completed: false},
				{ID: "step_4_3", Description: "修复发现的问题", Completed: false},
			},
		},
		{
			ID:          "phase_5",
			Name:        "Delivery",
			Description: "交付和总结",
			Status:      PhaseStatusPending,
			Order:       5,
			Steps: []TaskStep{
				{ID: "step_5_1", Description: "审查所有输出文件", Completed: false},
				{ID: "step_5_2", Description: "确保交付物完整", Completed: false},
				{ID: "step_5_3", Description: "交付给用户", Completed: false},
			},
		},
	}
}

// GetTask 获取任务
func (m *Manager) GetTask(ctx context.Context, taskID string) (*Task, error) {
	m.mu.RLock()
	if taskCtx, ok := m.activeTasks[taskID]; ok {
		m.mu.RUnlock()
		return taskCtx.Task, nil
	}
	m.mu.RUnlock()

	return m.storage.LoadTask(taskID)
}

// GetTaskContext 获取完整任务上下文
func (m *Manager) GetTaskContext(ctx context.Context, taskID string) (*TaskContext, error) {
	m.mu.RLock()
	if taskCtx, ok := m.activeTasks[taskID]; ok {
		m.mu.RUnlock()
		return taskCtx, nil
	}
	m.mu.RUnlock()

	return m.storage.LoadContext(taskID)
}

// UpdatePhaseStatus 更新阶段状态
// 遵循规则4：行动后更新（Update After Act）
func (m *Manager) UpdatePhaseStatus(ctx context.Context, taskID, phaseID string, status PhaseStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	now := time.Now()

	// 更新阶段状态
	for i := range taskCtx.Task.Phases {
		if taskCtx.Task.Phases[i].ID == phaseID {
			taskCtx.Task.Phases[i].Status = status
			if status == PhaseStatusInProgress && taskCtx.Task.Phases[i].StartedAt == nil {
				taskCtx.Task.Phases[i].StartedAt = &now
			}
			if status == PhaseStatusComplete {
				taskCtx.Task.Phases[i].CompletedAt = &now
			}
			break
		}
	}

	// 如果阶段完成，更新当前阶段为下一个
	if status == PhaseStatusComplete {
		for i := range taskCtx.Task.Phases {
			if taskCtx.Task.Phases[i].ID == phaseID {
				if i+1 < len(taskCtx.Task.Phases) {
					taskCtx.Task.CurrentPhase = taskCtx.Task.Phases[i+1].ID
					taskCtx.Task.Phases[i+1].Status = PhaseStatusInProgress
					startTime := time.Now()
					taskCtx.Task.Phases[i+1].StartedAt = &startTime
				}
				break
			}
		}
	}

	// 检查是否所有阶段都完成
	allComplete := true
	for _, phase := range taskCtx.Task.Phases {
		if phase.Status != PhaseStatusComplete {
			allComplete = false
			break
		}
	}

	if allComplete {
		taskCtx.Task.Status = TaskStatusCompleted
		completedAt := time.Now()
		taskCtx.Task.CompletedAt = &completedAt
	}

	// 记录进度
	taskCtx.Progress.Entries = append(taskCtx.Progress.Entries, ProgressEntry{
		PhaseID:   phaseID,
		Action:    fmt.Sprintf("Phase status updated to %s", status),
		Timestamp: now,
	})

	return m.storage.SaveContext(taskCtx)
}

// CompleteStep 完成步骤
func (m *Manager) CompleteStep(ctx context.Context, taskID, phaseID, stepID string, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	now := time.Now()

	// 更新步骤状态
	for i := range taskCtx.Task.Phases {
		if taskCtx.Task.Phases[i].ID == phaseID {
			for j := range taskCtx.Task.Phases[i].Steps {
				if taskCtx.Task.Phases[i].Steps[j].ID == stepID {
					taskCtx.Task.Phases[i].Steps[j].Completed = true
					taskCtx.Task.Phases[i].Steps[j].Result = result
					break
				}
			}

			// 检查该阶段是否所有步骤都完成
			allStepsComplete := true
			for _, step := range taskCtx.Task.Phases[i].Steps {
				if !step.Completed {
					allStepsComplete = false
					break
				}
			}

			if allStepsComplete {
				taskCtx.Task.Phases[i].Status = PhaseStatusComplete
				taskCtx.Task.Phases[i].CompletedAt = &now
			}
			break
		}
	}

	// 记录进度
	taskCtx.Progress.Entries = append(taskCtx.Progress.Entries, ProgressEntry{
		PhaseID:   phaseID,
		Action:    fmt.Sprintf("Completed step %s: %s", stepID, result),
		Timestamp: now,
	})

	return m.storage.SaveContext(taskCtx)
}

// RecordError 记录错误
// 遵循规则5：记录所有错误（Log ALL Errors）
func (m *Manager) RecordError(ctx context.Context, taskID string, errorMsg string, attempt int, resolution string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	now := time.Now()

	errorRecord := ErrorRecord{
		Error:      errorMsg,
		Attempt:    attempt,
		Resolution: resolution,
		Timestamp:  now,
		PhaseID:    taskCtx.Task.CurrentPhase,
	}

	// 添加到任务和进度的错误日志
	taskCtx.Task.Errors = append(taskCtx.Task.Errors, errorRecord)
	taskCtx.Progress.ErrorLog = append(taskCtx.Progress.ErrorLog, errorRecord)

	log.Warnf("Task %s error recorded: %s (attempt %d)", taskID, errorMsg, attempt)

	return m.storage.SaveContext(taskCtx)
}

// AddDecision 添加决策
func (m *Manager) AddDecision(ctx context.Context, taskID string, decision, rationale string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	taskCtx.Task.Decisions = append(taskCtx.Task.Decisions, Decision{
		Decision:  decision,
		Rationale: rationale,
		Timestamp: time.Now(),
		PhaseID:   taskCtx.Task.CurrentPhase,
	})

	return m.storage.SaveContext(taskCtx)
}

// AddFinding 添加发现
// 遵循2动作规则：每2次视图操作后保存发现
func (m *Manager) AddFinding(ctx context.Context, taskID string, category, content, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	taskCtx.Findings.Findings = append(taskCtx.Findings.Findings, Finding{
		Category:  category,
		Content:   content,
		Source:    source,
		Timestamp: time.Now(),
	})

	return m.storage.SaveContext(taskCtx)
}

// RecordViewAction 记录视图动作（用于2动作规则）
func (m *Manager) RecordViewAction(ctx context.Context, taskID string, actionType ActionType) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 只对视图类动作计数
	if actionType != ActionTypeView && actionType != ActionTypeBrowser && actionType != ActionTypeSearch {
		return false, nil
	}

	m.viewActionCount[taskID]++
	count := m.viewActionCount[taskID]

	// 每2次视图动作后需要保存发现
	needsSave := count >= 2
	if needsSave {
		m.viewActionCount[taskID] = 0
		log.Infof("Task %s: 2-action rule triggered, findings should be saved", taskID)
	}

	return needsSave, nil
}

// IncrementToolCallCount 增加工具调用计数
// 用于决定何时需要重读计划（注意力操纵）
func (m *Manager) IncrementToolCallCount(ctx context.Context, taskID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return false, err
	}

	taskCtx.Task.ToolCallCount++

	// 检查是否需要重读计划
	needsReread := taskCtx.Task.ToolCallCount >= m.config.RereadThreshold
	if needsReread {
		taskCtx.Task.ToolCallCount = 0
		taskCtx.Task.NeedsReread = true
		log.Infof("Task %s: Reread threshold reached, plan should be reread", taskID)
	}

	if err := m.storage.SaveTask(taskCtx.Task); err != nil {
		return false, err
	}

	return needsReread, nil
}

// GetTaskSummary 获取任务摘要（用于上下文压缩）
func (m *Manager) GetTaskSummary(ctx context.Context, taskID string) (*TaskSummary, error) {
	taskCtx, err := m.GetTaskContext(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if taskCtx == nil || taskCtx.Task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	task := taskCtx.Task

	// 收集已完成阶段
	var completedPhases []string
	for _, phase := range task.Phases {
		if phase.Status == PhaseStatusComplete {
			completedPhases = append(completedPhases, phase.Name)
		}
	}

	// 收集关键决策（最近5个）
	var keyDecisions []string
	start := 0
	if len(task.Decisions) > 5 {
		start = len(task.Decisions) - 5
	}
	for i := start; i < len(task.Decisions); i++ {
		keyDecisions = append(keyDecisions, task.Decisions[i].Decision)
	}

	// 收集最近错误（最近3个）
	var recentErrors []string
	errStart := 0
	if len(task.Errors) > 3 {
		errStart = len(task.Errors) - 3
	}
	for i := errStart; i < len(task.Errors); i++ {
		recentErrors = append(recentErrors, task.Errors[i].Error)
	}

	// 生成摘要文本
	summary := fmt.Sprintf("目标: %s\n当前阶段: %s\n已完成: %d/%d 阶段",
		task.Goal, task.CurrentPhase, len(completedPhases), len(task.Phases))

	return &TaskSummary{
		TaskID:          taskID,
		Goal:            task.Goal,
		CurrentPhase:    task.CurrentPhase,
		CompletedPhases: completedPhases,
		KeyDecisions:    keyDecisions,
		RecentErrors:    recentErrors,
		Summary:         summary,
	}, nil
}

// CheckCompletion 检查任务完成状态
// 遵循完成验证原则
func (m *Manager) CheckCompletion(ctx context.Context, taskID string) (bool, []string, error) {
	taskCtx, err := m.GetTaskContext(ctx, taskID)
	if err != nil {
		return false, nil, err
	}

	if taskCtx == nil || taskCtx.Task == nil {
		return false, nil, fmt.Errorf("task not found: %s", taskID)
	}

	var incompletePhases []string
	for _, phase := range taskCtx.Task.Phases {
		if phase.Status != PhaseStatusComplete {
			incompletePhases = append(incompletePhases, phase.Name)
		}
	}

	return len(incompletePhases) == 0, incompletePhases, nil
}

// getOrLoadContext 获取或加载任务上下文
func (m *Manager) getOrLoadContext(taskID string) (*TaskContext, error) {
	if taskCtx, ok := m.activeTasks[taskID]; ok {
		return taskCtx, nil
	}

	taskCtx, err := m.storage.LoadContext(taskID)
	if err != nil {
		return nil, err
	}

	if taskCtx == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	m.activeTasks[taskID] = taskCtx
	return taskCtx, nil
}

// StartPhase 开始阶段
func (m *Manager) StartPhase(ctx context.Context, taskID, phaseID string) error {
	return m.UpdatePhaseStatus(ctx, taskID, phaseID, PhaseStatusInProgress)
}

// ListTasks 列出任务
func (m *Manager) ListTasks(ctx context.Context, userID, sessionID string) ([]*Task, error) {
	return m.storage.ListTasks(userID, sessionID)
}

// DeleteTask 删除任务
func (m *Manager) DeleteTask(ctx context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.activeTasks, taskID)
	delete(m.viewActionCount, taskID)

	return m.storage.DeleteTask(taskID)
}

// MarkNeedsReread 标记需要重读计划
func (m *Manager) MarkNeedsReread(ctx context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	taskCtx.Task.NeedsReread = true
	return m.storage.SaveTask(taskCtx.Task)
}

// ClearNeedsReread 清除重读标记
func (m *Manager) ClearNeedsReread(ctx context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	taskCtx.Task.NeedsReread = false
	return m.storage.SaveTask(taskCtx.Task)
}

// AddResource 添加资源
func (m *Manager) AddResource(ctx context.Context, taskID, resource string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	taskCtx.Findings.Resources = append(taskCtx.Findings.Resources, resource)
	return m.storage.SaveFindings(taskCtx.Findings)
}

// AddTestResult 添加测试结果
func (m *Manager) AddTestResult(ctx context.Context, taskID string, result TestResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	taskCtx, err := m.getOrLoadContext(taskID)
	if err != nil {
		return err
	}

	taskCtx.Progress.TestResults = append(taskCtx.Progress.TestResults, result)
	return m.storage.SaveProgress(taskCtx.Progress)
}
