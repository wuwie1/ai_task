package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Service 任务服务
// 提供任务管理的核心业务逻辑
type Service struct {
	manager         *Manager
	planner         *Planner
	executor        *Executor
	contextEngineer *ContextEngineer
	sessions        map[string]*Session
	mu              sync.RWMutex
}

// NewService 创建任务服务
func NewService(config *TaskManagerConfig) (*Service, error) {
	if config == nil {
		config = DefaultTaskManagerConfig()
	}

	manager, err := NewManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create task manager: %w", err)
	}

	planner := NewPlanner()
	executor := NewExecutor(manager, nil)
	contextEngineer := NewContextEngineer(nil)

	return &Service{
		manager:         manager,
		planner:         planner,
		executor:        executor,
		contextEngineer: contextEngineer,
		sessions:        make(map[string]*Session),
	}, nil
}

// CreateTask 创建任务
func (s *Service) CreateTask(ctx context.Context, req *PlanRequest) (*PlanResponse, error) {
	// 使用 LLM 生成计划
	planResult, err := s.planner.GeneratePlan(ctx, req)
	if err != nil {
		log.Warnf("Failed to generate plan with LLM, using default: %v", err)
	}

	// 创建任务
	task, err := s.manager.CreateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 如果有 LLM 生成的计划，更新阶段
	if planResult != nil {
		phases := s.planner.ConvertToTaskPhases(planResult)
		if len(phases) > 0 {
			// 更新任务阶段
			taskCtx, _ := s.manager.GetTaskContext(ctx, task.ID)
			if taskCtx != nil {
				taskCtx.Task.Phases = phases
				taskCtx.Task.KeyQuestions = planResult.KeyQuestions
				taskCtx.Task.CurrentPhase = phases[0].ID
				_ = s.manager.storage.SaveTask(taskCtx.Task)
				task = taskCtx.Task
			}
		}
	}

	return &PlanResponse{
		TaskID:   task.ID,
		Goal:     task.Goal,
		Phases:   task.Phases,
		Estimate: "待评估",
	}, nil
}

// GetTask 获取任务
func (s *Service) GetTask(ctx context.Context, taskID string) (*Task, error) {
	return s.manager.GetTask(ctx, taskID)
}

// GetTaskContext 获取任务上下文
func (s *Service) GetTaskContext(ctx context.Context, taskID string) (*TaskContext, error) {
	return s.manager.GetTaskContext(ctx, taskID)
}

// ListTasks 列出任务
func (s *Service) ListTasks(ctx context.Context, userID, sessionID string) ([]*Task, error) {
	return s.manager.ListTasks(ctx, userID, sessionID)
}

// ExecuteTask 执行任务
func (s *Service) ExecuteTask(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	task, err := s.manager.GetTask(ctx, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if task == nil {
		return nil, fmt.Errorf("task not found: %s", req.TaskID)
	}

	// 如果指定了阶段，执行该阶段
	if req.PhaseID != "" {
		result, err := s.executor.ExecutePhase(ctx, req.TaskID, req.PhaseID)
		if err != nil {
			return nil, err
		}

		return &ExecuteResponse{
			TaskID:       req.TaskID,
			CurrentPhase: task.CurrentPhase,
			Status:       task.Status,
			Message:      result.Message,
			NextAction:   result.NextAction,
		}, nil
	}

	// 执行整个任务
	result, err := s.executor.ExecuteTask(ctx, req.TaskID)
	if err != nil {
		return nil, err
	}

	// 刷新任务状态
	task, _ = s.manager.GetTask(ctx, req.TaskID)

	return &ExecuteResponse{
		TaskID:       req.TaskID,
		CurrentPhase: task.CurrentPhase,
		Status:       task.Status,
		Message:      result.Message,
	}, nil
}

// UpdatePhase 更新阶段状态
func (s *Service) UpdatePhase(ctx context.Context, taskID, phaseID string, status PhaseStatus) error {
	return s.manager.UpdatePhaseStatus(ctx, taskID, phaseID, status)
}

// CompleteStep 完成步骤
func (s *Service) CompleteStep(ctx context.Context, taskID, phaseID, stepID, result string) error {
	return s.manager.CompleteStep(ctx, taskID, phaseID, stepID, result)
}

// AddFinding 添加发现
func (s *Service) AddFinding(ctx context.Context, taskID, category, content, source string) error {
	return s.manager.AddFinding(ctx, taskID, category, content, source)
}

// AddDecision 添加决策
func (s *Service) AddDecision(ctx context.Context, taskID, decision, rationale string) error {
	return s.manager.AddDecision(ctx, taskID, decision, rationale)
}

// RecordError 记录错误
func (s *Service) RecordError(ctx context.Context, taskID, errorMsg string, attempt int, resolution string) error {
	return s.manager.RecordError(ctx, taskID, errorMsg, attempt, resolution)
}

// CheckCompletion 检查完成状态
func (s *Service) CheckCompletion(ctx context.Context, taskID string) (*CompletionStatus, error) {
	checker := NewCompletionChecker(s.manager)
	return checker.Check(ctx, taskID)
}

// DeleteTask 删除任务
func (s *Service) DeleteTask(ctx context.Context, taskID string) error {
	return s.manager.DeleteTask(ctx, taskID)
}

// StartSession 开始会话
func (s *Service) StartSession(ctx context.Context, req *PlanRequest) (*SessionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := NewSession(s.manager)
	task, err := session.Start(ctx, req)
	if err != nil {
		return nil, err
	}

	s.sessions[session.ID] = session

	return &SessionInfo{
		SessionID: session.ID,
		TaskID:    task.ID,
		Goal:      task.Goal,
		Status:    string(task.Status),
		StartedAt: session.StartedAt,
	}, nil
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID string    `json:"session_id"`
	TaskID    string    `json:"task_id"`
	Goal      string    `json:"goal"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// GetSession 获取会话
func (s *Service) GetSession(sessionID string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	return session, ok
}

// ExecuteSession 执行会话
func (s *Service) ExecuteSession(ctx context.Context, sessionID string) (*ExecuteResponse, error) {
	session, ok := s.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	result, err := session.Execute(ctx)
	if err != nil {
		return nil, err
	}

	task, _ := s.manager.GetTask(ctx, session.TaskID)
	status := TaskStatusPending
	currentPhase := ""
	if task != nil {
		status = task.Status
		currentPhase = task.CurrentPhase
	}

	return &ExecuteResponse{
		TaskID:       session.TaskID,
		CurrentPhase: currentPhase,
		Status:       status,
		Message:      result.Message,
	}, nil
}

// CheckSessionStop 检查会话是否可以停止
func (s *Service) CheckSessionStop(ctx context.Context, sessionID string) (*CompletionStatus, error) {
	session, ok := s.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session.CheckStop(ctx)
}

// SessionPreAction 会话动作前钩子
func (s *Service) SessionPreAction(ctx context.Context, sessionID, actionName string) error {
	session, ok := s.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return session.PreAction(ctx, actionName)
}

// SessionPostAction 会话动作后钩子
func (s *Service) SessionPostAction(ctx context.Context, sessionID, actionName string, actionType ActionType) error {
	session, ok := s.GetSession(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	return session.PostAction(ctx, actionName, actionType)
}

// GetOptimizedContext 获取优化的上下文
func (s *Service) GetOptimizedContext(ctx context.Context, taskID string, toolCalls []ToolCall) (*OptimizedContext, error) {
	taskCtx, err := s.manager.GetTaskContext(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return s.contextEngineer.BuildOptimizedContext(ctx, taskCtx, toolCalls)
}

// GetTaskSummary 获取任务摘要
func (s *Service) GetTaskSummary(ctx context.Context, taskID string) (*TaskSummary, error) {
	return s.manager.GetTaskSummary(ctx, taskID)
}

// RefinePhase 细化阶段
func (s *Service) RefinePhase(ctx context.Context, taskID, phaseID string) ([]TaskStep, error) {
	taskCtx, err := s.manager.GetTaskContext(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return s.planner.RefinePhase(ctx, taskCtx, phaseID)
}

// AddResource 添加资源
func (s *Service) AddResource(ctx context.Context, taskID, resource string) error {
	return s.manager.AddResource(ctx, taskID, resource)
}

// AddTestResult 添加测试结果
func (s *Service) AddTestResult(ctx context.Context, taskID string, result TestResult) error {
	return s.manager.AddTestResult(ctx, taskID, result)
}

// RecordViewAction 记录视图动作
func (s *Service) RecordViewAction(ctx context.Context, taskID string, actionType ActionType) (bool, error) {
	return s.manager.RecordViewAction(ctx, taskID, actionType)
}
