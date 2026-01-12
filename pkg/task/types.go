// Package task 实现类似 Manus 的任务规划和执行系统
// 核心原则：
// 1. 文件系统作为外部记忆（持久化，无限）
// 2. 上下文窗口作为工作记忆（易失，有限）
// 3. 通过复述操纵注意力
// 4. 保留错误信息用于学习
// 5. KV 缓存优化
package task

import (
	"time"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"     // 待处理
	TaskStatusInProgress TaskStatus = "in_progress" // 进行中
	TaskStatusCompleted  TaskStatus = "completed"   // 已完成
	TaskStatusFailed     TaskStatus = "failed"      // 失败
	TaskStatusCancelled  TaskStatus = "cancelled"   // 已取消
)

// PhaseStatus 阶段状态
type PhaseStatus string

const (
	PhaseStatusPending    PhaseStatus = "pending"
	PhaseStatusInProgress PhaseStatus = "in_progress"
	PhaseStatusComplete   PhaseStatus = "complete"
	PhaseStatusFailed     PhaseStatus = "failed"
)

// TaskPhase 任务阶段
type TaskPhase struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Status      PhaseStatus `json:"status"`
	Steps       []TaskStep  `json:"steps"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Order       int         `json:"order"`
}

// TaskStep 任务步骤
type TaskStep struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	Result      string `json:"result,omitempty"`
}

// ErrorRecord 错误记录
type ErrorRecord struct {
	Error      string    `json:"error"`
	Attempt    int       `json:"attempt"`
	Resolution string    `json:"resolution,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	PhaseID    string    `json:"phase_id,omitempty"`
}

// Decision 决策记录
type Decision struct {
	Decision  string    `json:"decision"`
	Rationale string    `json:"rationale"`
	Timestamp time.Time `json:"timestamp"`
	PhaseID   string    `json:"phase_id,omitempty"`
}

// Finding 发现记录
type Finding struct {
	Category  string    `json:"category"` // research, technical, visual, resource
	Content   string    `json:"content"`
	Source    string    `json:"source,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ProgressEntry 进度条目
type ProgressEntry struct {
	PhaseID   string    `json:"phase_id"`
	Action    string    `json:"action"`
	Files     []string  `json:"files,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// TestResult 测试结果
type TestResult struct {
	Test     string `json:"test"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Status   string `json:"status"` // ✓, ✗, pending
}

// Task 任务（对应 task_plan.md）
type Task struct {
	ID           string        `json:"id"`
	UserID       string        `json:"user_id"`
	SessionID    string        `json:"session_id"`
	Goal         string        `json:"goal"`
	CurrentPhase string        `json:"current_phase"`
	Phases       []TaskPhase   `json:"phases"`
	KeyQuestions []string      `json:"key_questions,omitempty"`
	Decisions    []Decision    `json:"decisions,omitempty"`
	Errors       []ErrorRecord `json:"errors,omitempty"`
	Status       TaskStatus    `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`

	// 上下文管理元数据
	ToolCallCount int  `json:"tool_call_count"` // 工具调用计数，用于决定何时重读计划
	NeedsReread   bool `json:"needs_reread"`    // 标记是否需要重读计划
}

// TaskFindings 任务发现（对应 findings.md）
type TaskFindings struct {
	TaskID       string    `json:"task_id"`
	Requirements []string  `json:"requirements,omitempty"`
	Findings     []Finding `json:"findings,omitempty"`
	Resources    []string  `json:"resources,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TaskProgress 任务进度（对应 progress.md）
type TaskProgress struct {
	TaskID      string          `json:"task_id"`
	SessionDate string          `json:"session_date"`
	Entries     []ProgressEntry `json:"entries,omitempty"`
	TestResults []TestResult    `json:"test_results,omitempty"`
	ErrorLog    []ErrorRecord   `json:"error_log,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// TaskContext 任务上下文（包含任务、发现和进度）
type TaskContext struct {
	Task     *Task         `json:"task"`
	Findings *TaskFindings `json:"findings"`
	Progress *TaskProgress `json:"progress"`
}

// ContextCompression 上下文压缩配置
type ContextCompression struct {
	MaxToolResultsInContext int  `json:"max_tool_results_in_context"` // 上下文中保留的最大工具结果数
	CompressOlderResults    bool `json:"compress_older_results"`      // 是否压缩较旧的结果
	KeepReferencesOnly      bool `json:"keep_references_only"`        // 只保留引用（文件路径、URL等）
}

// TaskManagerConfig 任务管理器配置
type TaskManagerConfig struct {
	StoragePath          string             `json:"storage_path"`           // 任务文件存储路径
	RereadThreshold      int                `json:"reread_threshold"`       // 重读计划的工具调用阈值（默认10）
	TwoActionRuleEnabled bool               `json:"two_action_rule"`        // 是否启用2动作规则
	Compression          ContextCompression `json:"compression"`            // 上下文压缩配置
	MaxRetries           int                `json:"max_retries"`            // 最大重试次数（3次打击规则）
	EnableAutoPlanning   bool               `json:"enable_auto_planning"`   // 是否启用自动规划
}

// DefaultTaskManagerConfig 返回默认配置
func DefaultTaskManagerConfig() *TaskManagerConfig {
	return &TaskManagerConfig{
		StoragePath:          ".tasks",
		RereadThreshold:      10,
		TwoActionRuleEnabled: true,
		Compression: ContextCompression{
			MaxToolResultsInContext: 5,
			CompressOlderResults:    true,
			KeepReferencesOnly:      false,
		},
		MaxRetries:         3,
		EnableAutoPlanning: true,
	}
}

// ToolCall 工具调用记录
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Args      map[string]interface{} `json:"args,omitempty"`
	Result    string                 `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Compressed bool                  `json:"compressed"` // 是否已压缩
}

// ActionType 动作类型（用于2动作规则）
type ActionType string

const (
	ActionTypeView    ActionType = "view"    // 查看类操作
	ActionTypeBrowser ActionType = "browser" // 浏览器操作
	ActionTypeSearch  ActionType = "search"  // 搜索操作
	ActionTypeWrite   ActionType = "write"   // 写入操作
	ActionTypeExecute ActionType = "execute" // 执行操作
)

// PlanRequest 规划请求
type PlanRequest struct {
	UserID      string   `json:"user_id" binding:"required"`
	SessionID   string   `json:"session_id" binding:"required"`
	Goal        string   `json:"goal" binding:"required"`
	Context     string   `json:"context,omitempty"`      // 额外上下文信息
	Constraints []string `json:"constraints,omitempty"`  // 约束条件
	Preferences []string `json:"preferences,omitempty"`  // 偏好设置
}

// PlanResponse 规划响应
type PlanResponse struct {
	TaskID   string      `json:"task_id"`
	Goal     string      `json:"goal"`
	Phases   []TaskPhase `json:"phases"`
	Estimate string      `json:"estimate,omitempty"` // 预估完成时间
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	TaskID  string `json:"task_id" binding:"required"`
	PhaseID string `json:"phase_id,omitempty"` // 可选，指定执行特定阶段
	StepID  string `json:"step_id,omitempty"`  // 可选，指定执行特定步骤
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	TaskID       string     `json:"task_id"`
	CurrentPhase string     `json:"current_phase"`
	Status       TaskStatus `json:"status"`
	Message      string     `json:"message"`
	NextAction   string     `json:"next_action,omitempty"` // 建议的下一步操作
}

// TaskSummary 任务摘要（用于上下文压缩）
type TaskSummary struct {
	TaskID         string   `json:"task_id"`
	Goal           string   `json:"goal"`
	CurrentPhase   string   `json:"current_phase"`
	CompletedPhases []string `json:"completed_phases"`
	KeyDecisions   []string `json:"key_decisions"`
	RecentErrors   []string `json:"recent_errors"`
	Summary        string   `json:"summary"`
}
