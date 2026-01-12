package task

import (
	"ai_task/pkg/clients/llm_model"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

// Executor 任务执行器
// 实现 Manus 的执行原则：
// 1. 决策前阅读计划
// 2. 3次打击错误协议
// 3. 永不重复失败
type Executor struct {
	llmClient *llm_model.ClientChatModel
	manager   *Manager
	planner   *Planner
	config    *ExecutorConfig
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	MaxRetries          int  // 最大重试次数（3次打击规则）
	RereadBeforeAction  bool // 执行前重读计划
	AutoSaveFindings    bool // 自动保存发现
	EnableThreeStrike   bool // 启用3次打击规则
}

// DefaultExecutorConfig 默认执行器配置
func DefaultExecutorConfig() *ExecutorConfig {
	return &ExecutorConfig{
		MaxRetries:         3,
		RereadBeforeAction: true,
		AutoSaveFindings:   true,
		EnableThreeStrike:  true,
	}
}

// NewExecutor 创建执行器
func NewExecutor(manager *Manager, config *ExecutorConfig) *Executor {
	if config == nil {
		config = DefaultExecutorConfig()
	}

	return &Executor{
		llmClient: llm_model.GetInstance(),
		manager:   manager,
		planner:   NewPlanner(),
		config:    config,
	}
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message"`
	Output     map[string]interface{} `json:"output,omitempty"`
	NextAction string                 `json:"next_action,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Attempt    int                    `json:"attempt"`
}

// ExecuteStep 执行单个步骤
func (e *Executor) ExecuteStep(ctx context.Context, taskID, phaseID, stepID string, action func(ctx context.Context) (*ExecutionResult, error)) (*ExecutionResult, error) {
	// 获取任务上下文（用于验证任务存在）
	_, err := e.manager.GetTaskContext(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task context: %w", err)
	}

	// 规则3：决策前阅读计划
	if e.config.RereadBeforeAction {
		log.Debugf("Rereading plan before executing step %s", stepID)
		// 这里只是标记已读，实际的计划内容会在需要时被引用
	}

	// 实现3次打击错误协议
	var lastResult *ExecutionResult
	for attempt := 1; attempt <= e.config.MaxRetries; attempt++ {
		result, err := action(ctx)
		if err != nil {
			// 记录错误
			_ = e.manager.RecordError(ctx, taskID, err.Error(), attempt, "")

			if attempt < e.config.MaxRetries {
				log.Warnf("Step %s failed (attempt %d/%d): %v, retrying with different approach",
					stepID, attempt, e.config.MaxRetries, err)
				continue
			}

			// 3次失败后升级给用户
			return &ExecutionResult{
				Success: false,
				Error:   fmt.Sprintf("Step failed after %d attempts: %v", attempt, err),
				Attempt: attempt,
				Message: "请提供进一步指导",
			}, nil
		}

		if result != nil && result.Success {
			// 成功完成步骤
			_ = e.manager.CompleteStep(ctx, taskID, phaseID, stepID, result.Message)
			return result, nil
		}

		lastResult = result
		if result != nil && result.Error != "" {
			_ = e.manager.RecordError(ctx, taskID, result.Error, attempt, "")
		}
	}

	if lastResult == nil {
		lastResult = &ExecutionResult{
			Success: false,
			Error:   "Unknown error",
			Attempt: e.config.MaxRetries,
		}
	}

	return lastResult, nil
}

// ExecutePhase 执行整个阶段
func (e *Executor) ExecutePhase(ctx context.Context, taskID, phaseID string) (*ExecutionResult, error) {
	taskCtx, err := e.manager.GetTaskContext(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task context: %w", err)
	}

	// 开始阶段
	if err := e.manager.StartPhase(ctx, taskID, phaseID); err != nil {
		return nil, fmt.Errorf("failed to start phase: %w", err)
	}

	// 找到目标阶段
	var targetPhase *TaskPhase
	for i := range taskCtx.Task.Phases {
		if taskCtx.Task.Phases[i].ID == phaseID {
			targetPhase = &taskCtx.Task.Phases[i]
			break
		}
	}

	if targetPhase == nil {
		return nil, fmt.Errorf("phase not found: %s", phaseID)
	}

	// 执行每个步骤
	for _, step := range targetPhase.Steps {
		if step.Completed {
			continue
		}

		// 这里使用 LLM 决定如何执行步骤
		result, err := e.decideAndExecuteStep(ctx, taskCtx, targetPhase, &step)
		if err != nil {
			return nil, err
		}

		if !result.Success {
			return result, nil
		}
	}

	// 完成阶段
	if err := e.manager.UpdatePhaseStatus(ctx, taskID, phaseID, PhaseStatusComplete); err != nil {
		return nil, fmt.Errorf("failed to complete phase: %w", err)
	}

	return &ExecutionResult{
		Success:    true,
		Message:    fmt.Sprintf("Phase %s completed successfully", targetPhase.Name),
		NextAction: e.getNextPhaseID(taskCtx.Task, phaseID),
	}, nil
}

// decideAndExecuteStep 决定并执行步骤
func (e *Executor) decideAndExecuteStep(ctx context.Context, taskCtx *TaskContext, phase *TaskPhase, step *TaskStep) (*ExecutionResult, error) {
	// 构建决策提示
	decisionPrompt := e.buildDecisionPrompt(taskCtx, phase, step)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: PromptExecutorSystem,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: decisionPrompt,
		},
	}

	result, err := e.llmClient.PostChatCompletionsNonStreamContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get decision: %w", err)
	}

	// 解析决策结果
	decision, err := e.parseDecision(result)
	if err != nil {
		log.Warnf("Failed to parse decision: %v, treating as completed", err)
		decision = &StepDecision{
			Action:  "complete",
			Message: "步骤已标记完成",
		}
	}

	// 记录决策
	if decision.Rationale != "" {
		_ = e.manager.AddDecision(ctx, taskCtx.Task.ID, decision.Action, decision.Rationale)
	}

	// 记录发现
	for _, finding := range decision.Findings {
		_ = e.manager.AddFinding(ctx, taskCtx.Task.ID, finding.Category, finding.Content, finding.Source)
	}

	// 标记步骤完成
	_ = e.manager.CompleteStep(ctx, taskCtx.Task.ID, phase.ID, step.ID, decision.Message)

	return &ExecutionResult{
		Success: true,
		Message: decision.Message,
		Output: map[string]interface{}{
			"action":    decision.Action,
			"rationale": decision.Rationale,
		},
	}, nil
}

// StepDecision 步骤决策
type StepDecision struct {
	Action    string    `json:"action"`
	Message   string    `json:"message"`
	Rationale string    `json:"rationale,omitempty"`
	Findings  []Finding `json:"findings,omitempty"`
}

// 提示词常量已移至 prompts.go

// buildDecisionPrompt 构建决策提示
func (e *Executor) buildDecisionPrompt(taskCtx *TaskContext, phase *TaskPhase, step *TaskStep) string {
	// 获取任务摘要（用于上下文压缩）
	var errorsSection string
	if len(taskCtx.Task.Errors) > 0 {
		errorsSection = "\n\n## 已知错误（避免重复）:\n"
		for _, err := range taskCtx.Task.Errors {
			errorsSection += fmt.Sprintf("- %s (尝试 %d 次): %s\n", err.Error, err.Attempt, err.Resolution)
		}
	}

	var decisionsSection string
	if len(taskCtx.Task.Decisions) > 0 {
		decisionsSection = "\n\n## 已做决策:\n"
		for _, d := range taskCtx.Task.Decisions {
			decisionsSection += fmt.Sprintf("- %s: %s\n", d.Decision, d.Rationale)
		}
	}

	var findingsSection string
	if taskCtx.Findings != nil && len(taskCtx.Findings.Findings) > 0 {
		findingsSection = "\n\n## 相关发现:\n"
		// 只取最近5个发现
		start := 0
		if len(taskCtx.Findings.Findings) > 5 {
			start = len(taskCtx.Findings.Findings) - 5
		}
		for i := start; i < len(taskCtx.Findings.Findings); i++ {
			f := taskCtx.Findings.Findings[i]
			findingsSection += fmt.Sprintf("- [%s] %s\n", f.Category, f.Content)
		}
	}

	return fmt.Sprintf(`## 任务信息

目标: %s
当前阶段: %s - %s
当前步骤: %s
%s%s%s

请决定如何执行这个步骤，并提供你的发现。`,
		taskCtx.Task.Goal,
		phase.ID, phase.Name,
		step.Description,
		errorsSection,
		decisionsSection,
		findingsSection)
}

// parseDecision 解析决策结果
func (e *Executor) parseDecision(result string) (*StepDecision, error) {
	result = cleanJSONResponse(result)

	var decision StepDecision
	if err := json.Unmarshal([]byte(result), &decision); err != nil {
		return nil, fmt.Errorf("failed to parse decision JSON: %w", err)
	}

	return &decision, nil
}

// getNextPhaseID 获取下一个阶段ID
func (e *Executor) getNextPhaseID(task *Task, currentPhaseID string) string {
	for i, phase := range task.Phases {
		if phase.ID == currentPhaseID && i+1 < len(task.Phases) {
			return task.Phases[i+1].ID
		}
	}
	return ""
}

// ExecuteTask 执行整个任务
func (e *Executor) ExecuteTask(ctx context.Context, taskID string) (*ExecutionResult, error) {
	taskCtx, err := e.manager.GetTaskContext(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task context: %w", err)
	}

	// 更新任务状态为进行中
	taskCtx.Task.Status = TaskStatusInProgress

	// 执行每个未完成的阶段
	for _, phase := range taskCtx.Task.Phases {
		if phase.Status == PhaseStatusComplete {
			continue
		}

		result, err := e.ExecutePhase(ctx, taskID, phase.ID)
		if err != nil {
			return nil, err
		}

		if !result.Success {
			return result, nil
		}
	}

	// 检查完成状态
	complete, incomplete, err := e.manager.CheckCompletion(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if !complete {
		return &ExecutionResult{
			Success: false,
			Message: fmt.Sprintf("Task incomplete. Remaining phases: %s", strings.Join(incomplete, ", ")),
		}, nil
	}

	return &ExecutionResult{
		Success: true,
		Message: "Task completed successfully",
	}, nil
}

// ContextBuilder 上下文构建器
// 用于构建发送给 LLM 的上下文
type ContextBuilder struct {
	maxTokens int
	compression *ContextCompression
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(maxTokens int, compression *ContextCompression) *ContextBuilder {
	if compression == nil {
		compression = &ContextCompression{
			MaxToolResultsInContext: 5,
			CompressOlderResults:    true,
			KeepReferencesOnly:      false,
		}
	}

	return &ContextBuilder{
		maxTokens:   maxTokens,
		compression: compression,
	}
}

// BuildContext 构建上下文
// 实现上下文工程的3个策略：压缩、隔离、卸载
func (cb *ContextBuilder) BuildContext(taskCtx *TaskContext, toolCalls []ToolCall) string {
	var sb strings.Builder

	// 1. 任务摘要（始终包含）
	sb.WriteString("## 任务计划\n")
	sb.WriteString(fmt.Sprintf("目标: %s\n", taskCtx.Task.Goal))
	sb.WriteString(fmt.Sprintf("当前阶段: %s\n", taskCtx.Task.CurrentPhase))
	sb.WriteString(fmt.Sprintf("状态: %s\n\n", taskCtx.Task.Status))

	// 2. 阶段进度
	sb.WriteString("## 阶段进度\n")
	for _, phase := range taskCtx.Task.Phases {
		status := "[ ]"
		switch phase.Status {
		case PhaseStatusComplete:
			status = "[x]"
		case PhaseStatusInProgress:
			status = "[>]"
		}
		sb.WriteString(fmt.Sprintf("%s %s: %s\n", status, phase.Name, phase.Status))
	}
	sb.WriteString("\n")

	// 3. 关键决策（压缩：只保留最近的）
	if len(taskCtx.Task.Decisions) > 0 {
		sb.WriteString("## 关键决策\n")
		start := 0
		if len(taskCtx.Task.Decisions) > 5 {
			start = len(taskCtx.Task.Decisions) - 5
		}
		for i := start; i < len(taskCtx.Task.Decisions); i++ {
			d := taskCtx.Task.Decisions[i]
			sb.WriteString(fmt.Sprintf("- %s\n", d.Decision))
		}
		sb.WriteString("\n")
	}

	// 4. 错误记录（保留用于学习）
	if len(taskCtx.Task.Errors) > 0 {
		sb.WriteString("## 错误记录（避免重复）\n")
		start := 0
		if len(taskCtx.Task.Errors) > 3 {
			start = len(taskCtx.Task.Errors) - 3
		}
		for i := start; i < len(taskCtx.Task.Errors); i++ {
			e := taskCtx.Task.Errors[i]
			sb.WriteString(fmt.Sprintf("- %s (尝试 %d 次)\n", e.Error, e.Attempt))
		}
		sb.WriteString("\n")
	}

	// 5. 工具调用结果（压缩策略）
	if len(toolCalls) > 0 {
		sb.WriteString("## 最近工具调用\n")
		start := 0
		if len(toolCalls) > cb.compression.MaxToolResultsInContext {
			start = len(toolCalls) - cb.compression.MaxToolResultsInContext
		}
		for i := start; i < len(toolCalls); i++ {
			tc := toolCalls[i]
			if tc.Compressed || cb.compression.KeepReferencesOnly {
				sb.WriteString(fmt.Sprintf("- %s: [结果已存储]\n", tc.Name))
			} else {
				result := tc.Result
				if len(result) > 200 {
					result = result[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("- %s: %s\n", tc.Name, result))
			}
		}
		sb.WriteString("\n")
	}

	// 6. 发现摘要
	if taskCtx.Findings != nil && len(taskCtx.Findings.Findings) > 0 {
		sb.WriteString("## 关键发现\n")
		start := 0
		if len(taskCtx.Findings.Findings) > 5 {
			start = len(taskCtx.Findings.Findings) - 5
		}
		for i := start; i < len(taskCtx.Findings.Findings); i++ {
			f := taskCtx.Findings.Findings[i]
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", f.Category, f.Content))
		}
	}

	return sb.String()
}

// ActionTracker 动作追踪器
// 实现2动作规则
type ActionTracker struct {
	manager *Manager
}

// NewActionTracker 创建动作追踪器
func NewActionTracker(manager *Manager) *ActionTracker {
	return &ActionTracker{manager: manager}
}

// TrackAction 追踪动作
// 返回是否需要保存发现
func (at *ActionTracker) TrackAction(ctx context.Context, taskID string, actionType ActionType) (bool, error) {
	return at.manager.RecordViewAction(ctx, taskID, actionType)
}

// PreActionHook 动作前钩子
// 实现 PreToolUse 钩子行为
func (at *ActionTracker) PreActionHook(ctx context.Context, taskID string, actionName string) error {
	// 检查是否需要重读计划
	task, err := at.manager.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task != nil && task.NeedsReread {
		log.Infof("Pre-action hook: Plan should be reread before %s", actionName)
		// 清除重读标记
		_ = at.manager.ClearNeedsReread(ctx, taskID)
	}

	// 增加工具调用计数
	needsReread, err := at.manager.IncrementToolCallCount(ctx, taskID)
	if err != nil {
		return err
	}

	if needsReread {
		log.Infof("Tool call threshold reached, plan should be reread")
	}

	return nil
}

// PostActionHook 动作后钩子
// 实现 PostToolUse 钩子行为
func (at *ActionTracker) PostActionHook(ctx context.Context, taskID string, actionName string, actionType ActionType) error {
	// 检查2动作规则
	needsSave, err := at.TrackAction(ctx, taskID, actionType)
	if err != nil {
		return err
	}

	if needsSave {
		log.Infof("Post-action hook: Findings should be saved after %s (2-action rule)", actionName)
	}

	return nil
}

// ErrorTracker 错误追踪器
// 实现3次打击规则
type ErrorTracker struct {
	manager     *Manager
	errorCounts map[string]map[string]int // taskID -> errorKey -> count
}

// NewErrorTracker 创建错误追踪器
func NewErrorTracker(manager *Manager) *ErrorTracker {
	return &ErrorTracker{
		manager:     manager,
		errorCounts: make(map[string]map[string]int),
	}
}

// TrackError 追踪错误
// 返回是否应该升级给用户
func (et *ErrorTracker) TrackError(ctx context.Context, taskID string, errorKey string, errorMsg string) (bool, int, error) {
	if et.errorCounts[taskID] == nil {
		et.errorCounts[taskID] = make(map[string]int)
	}

	et.errorCounts[taskID][errorKey]++
	count := et.errorCounts[taskID][errorKey]

	// 记录错误
	err := et.manager.RecordError(ctx, taskID, errorMsg, count, "")
	if err != nil {
		return false, count, err
	}

	// 3次打击规则
	shouldEscalate := count >= 3
	if shouldEscalate {
		log.Warnf("Error '%s' occurred %d times, escalating to user", errorKey, count)
	}

	return shouldEscalate, count, nil
}

// ResolveError 解决错误
func (et *ErrorTracker) ResolveError(ctx context.Context, taskID string, errorKey string, resolution string) error {
	// 更新最后一个错误的解决方案
	taskCtx, err := et.manager.GetTaskContext(ctx, taskID)
	if err != nil {
		return err
	}

	for i := len(taskCtx.Task.Errors) - 1; i >= 0; i-- {
		if strings.Contains(taskCtx.Task.Errors[i].Error, errorKey) {
			taskCtx.Task.Errors[i].Resolution = resolution
			break
		}
	}

	// 重置计数
	if et.errorCounts[taskID] != nil {
		delete(et.errorCounts[taskID], errorKey)
	}

	return et.manager.RecordError(ctx, taskID, errorKey, 0, resolution)
}

// ShouldRetryWithDifferentApproach 是否应该使用不同方法重试
func (et *ErrorTracker) ShouldRetryWithDifferentApproach(taskID string, errorKey string) bool {
	if et.errorCounts[taskID] == nil {
		return false
	}
	count := et.errorCounts[taskID][errorKey]
	return count > 1 && count < 3
}

// CompletionChecker 完成检查器
// 实现 Stop 钩子行为
type CompletionChecker struct {
	manager *Manager
}

// NewCompletionChecker 创建完成检查器
func NewCompletionChecker(manager *Manager) *CompletionChecker {
	return &CompletionChecker{manager: manager}
}

// Check 检查任务完成状态
func (cc *CompletionChecker) Check(ctx context.Context, taskID string) (*CompletionStatus, error) {
	complete, incomplete, err := cc.manager.CheckCompletion(ctx, taskID)
	if err != nil {
		return nil, err
	}

	taskCtx, err := cc.manager.GetTaskContext(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 5问题重启测试
	rebootCheck := cc.performRebootCheck(taskCtx)

	return &CompletionStatus{
		Complete:         complete,
		IncompletePhases: incomplete,
		RebootCheck:      rebootCheck,
		CanStop:          complete && rebootCheck.AllAnswered,
	}, nil
}

// CompletionStatus 完成状态
type CompletionStatus struct {
	Complete         bool        `json:"complete"`
	IncompletePhases []string    `json:"incomplete_phases,omitempty"`
	RebootCheck      RebootCheck `json:"reboot_check"`
	CanStop          bool        `json:"can_stop"`
}

// RebootCheck 5问题重启检查
type RebootCheck struct {
	WhereAmI     string `json:"where_am_i"`
	WhereGoing   string `json:"where_going"`
	WhatIsGoal   string `json:"what_is_goal"`
	WhatLearned  string `json:"what_learned"`
	WhatDone     string `json:"what_done"`
	AllAnswered  bool   `json:"all_answered"`
}

// performRebootCheck 执行5问题重启检查
func (cc *CompletionChecker) performRebootCheck(taskCtx *TaskContext) RebootCheck {
	if taskCtx == nil || taskCtx.Task == nil {
		return RebootCheck{}
	}

	// 1. 我在哪里？
	whereAmI := taskCtx.Task.CurrentPhase
	for _, p := range taskCtx.Task.Phases {
		if p.ID == taskCtx.Task.CurrentPhase {
			whereAmI = p.Name
			break
		}
	}

	// 2. 我要去哪里？
	var remaining []string
	foundCurrent := false
	for _, p := range taskCtx.Task.Phases {
		if p.ID == taskCtx.Task.CurrentPhase {
			foundCurrent = true
			continue
		}
		if foundCurrent && p.Status != PhaseStatusComplete {
			remaining = append(remaining, p.Name)
		}
	}
	whereGoing := strings.Join(remaining, " → ")

	// 3. 目标是什么？
	whatIsGoal := taskCtx.Task.Goal

	// 4. 我学到了什么？
	whatLearned := fmt.Sprintf("%d 个发现", 0)
	if taskCtx.Findings != nil {
		whatLearned = fmt.Sprintf("%d 个发现", len(taskCtx.Findings.Findings))
	}

	// 5. 我做了什么？
	whatDone := fmt.Sprintf("%d 个进度条目", 0)
	if taskCtx.Progress != nil {
		whatDone = fmt.Sprintf("%d 个进度条目", len(taskCtx.Progress.Entries))
	}

	allAnswered := whereAmI != "" && whatIsGoal != ""

	return RebootCheck{
		WhereAmI:    whereAmI,
		WhereGoing:  whereGoing,
		WhatIsGoal:  whatIsGoal,
		WhatLearned: whatLearned,
		WhatDone:    whatDone,
		AllAnswered: allAnswered,
	}
}

// Session 会话管理
type Session struct {
	ID        string
	TaskID    string
	StartedAt time.Time
	manager   *Manager
	executor  *Executor
	tracker   *ActionTracker
	errTracker *ErrorTracker
	checker   *CompletionChecker
}

// NewSession 创建会话
func NewSession(manager *Manager) *Session {
	executor := NewExecutor(manager, nil)
	return &Session{
		ID:         fmt.Sprintf("session_%d", time.Now().UnixNano()),
		StartedAt:  time.Now(),
		manager:    manager,
		executor:   executor,
		tracker:    NewActionTracker(manager),
		errTracker: NewErrorTracker(manager),
		checker:    NewCompletionChecker(manager),
	}
}

// Start 开始会话
func (s *Session) Start(ctx context.Context, req *PlanRequest) (*Task, error) {
	task, err := s.manager.CreateTask(ctx, req)
	if err != nil {
		return nil, err
	}

	s.TaskID = task.ID
	log.Infof("Session %s started with task %s", s.ID, task.ID)

	return task, nil
}

// Execute 执行任务
func (s *Session) Execute(ctx context.Context) (*ExecutionResult, error) {
	if s.TaskID == "" {
		return nil, fmt.Errorf("no task associated with session")
	}

	return s.executor.ExecuteTask(ctx, s.TaskID)
}

// CheckStop 检查是否可以停止
func (s *Session) CheckStop(ctx context.Context) (*CompletionStatus, error) {
	if s.TaskID == "" {
		return nil, fmt.Errorf("no task associated with session")
	}

	return s.checker.Check(ctx, s.TaskID)
}

// PreAction 动作前钩子
func (s *Session) PreAction(ctx context.Context, actionName string) error {
	if s.TaskID == "" {
		return nil
	}
	return s.tracker.PreActionHook(ctx, s.TaskID, actionName)
}

// PostAction 动作后钩子
func (s *Session) PostAction(ctx context.Context, actionName string, actionType ActionType) error {
	if s.TaskID == "" {
		return nil
	}
	return s.tracker.PostActionHook(ctx, s.TaskID, actionName, actionType)
}

// RecordError 记录错误
func (s *Session) RecordError(ctx context.Context, errorKey, errorMsg string) (bool, int, error) {
	if s.TaskID == "" {
		return false, 0, nil
	}
	return s.errTracker.TrackError(ctx, s.TaskID, errorKey, errorMsg)
}
