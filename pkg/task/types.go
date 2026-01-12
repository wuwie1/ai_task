// Package task 实现类似 Manus 的任务规划和执行系统
// 核心原则：
// 1. 文件系统作为外部记忆（持久化，无限）
// 2. 上下文窗口作为工作记忆（易失，有限）
// 3. 通过复述操纵注意力
// 4. 保留错误信息用于学习
// 5. KV 缓存优化
package task

import (
	"ai_task/constant"
	"time"
)

// 类型别名，方便包内使用
type (
	TaskStatus  = constant.TaskStatus
	PhaseStatus = constant.PhaseStatus
	StorageType = constant.StorageType
	ActionType  = constant.ActionType
)

// 常量别名，保持向后兼容
const (
	TaskStatusPending    = constant.TaskStatusPending
	TaskStatusInProgress = constant.TaskStatusInProgress
	TaskStatusCompleted  = constant.TaskStatusCompleted
	TaskStatusFailed     = constant.TaskStatusFailed
	TaskStatusCancelled  = constant.TaskStatusCancelled

	PhaseStatusPending    = constant.PhaseStatusPending
	PhaseStatusInProgress = constant.PhaseStatusInProgress
	PhaseStatusComplete   = constant.PhaseStatusComplete
	PhaseStatusFailed     = constant.PhaseStatusFailed

	StorageTypeFile   = constant.StorageTypeFile
	StorageTypeDB     = constant.StorageTypeDB
	StorageTypeHybrid = constant.StorageTypeHybrid

	ActionTypeView    = constant.ActionTypeView
	ActionTypeBrowser = constant.ActionTypeBrowser
	ActionTypeSearch  = constant.ActionTypeSearch
	ActionTypeWrite   = constant.ActionTypeWrite
	ActionTypeExecute = constant.ActionTypeExecute
)

// TaskPhase 任务阶段
type TaskPhase struct {
	ID          string      `json:"id"`                     // 阶段唯一标识符
	Name        string      `json:"name"`                   // 阶段名称
	Description string      `json:"description"`            // 阶段描述
	Status      PhaseStatus `json:"status"`                 // 阶段状态（pending/in_progress/complete/failed）
	Steps       []TaskStep  `json:"steps"`                  // 阶段包含的步骤列表
	StartedAt   *time.Time  `json:"started_at,omitempty"`   // 阶段开始时间
	CompletedAt *time.Time  `json:"completed_at,omitempty"` // 阶段完成时间
	Order       int         `json:"order"`                  // 阶段执行顺序
}

// TaskStep 任务步骤
type TaskStep struct {
	ID          string `json:"id"`               // 步骤唯一标识符
	Description string `json:"description"`      // 步骤描述
	Completed   bool   `json:"completed"`        // 是否已完成
	Result      string `json:"result,omitempty"` // 步骤执行结果
}

// ErrorRecord 错误记录
type ErrorRecord struct {
	Error      string    `json:"error"`                // 错误信息
	Attempt    int       `json:"attempt"`              // 尝试次数
	Resolution string    `json:"resolution,omitempty"` // 解决方案（如果有）
	Timestamp  time.Time `json:"timestamp"`            // 错误发生时间
	PhaseID    string    `json:"phase_id,omitempty"`   // 关联的阶段ID
}

// Decision 决策记录
type Decision struct {
	Decision  string    `json:"decision"`           // 决策内容
	Rationale string    `json:"rationale"`          // 决策理由
	Timestamp time.Time `json:"timestamp"`          // 决策时间
	PhaseID   string    `json:"phase_id,omitempty"` // 关联的阶段ID
}

// Finding 发现记录
type Finding struct {
	Category  string    `json:"category"`         // 发现类别（research/technical/visual/resource）
	Content   string    `json:"content"`          // 发现内容
	Source    string    `json:"source,omitempty"` // 发现来源（文件路径、URL等）
	Timestamp time.Time `json:"timestamp"`        // 发现时间
}

// ProgressEntry 进度条目
type ProgressEntry struct {
	PhaseID   string    `json:"phase_id"`        // 关联的阶段ID
	Action    string    `json:"action"`          // 执行的操作描述
	Files     []string  `json:"files,omitempty"` // 涉及的文件列表
	Timestamp time.Time `json:"timestamp"`       // 操作时间
}

// TestResult 测试结果
type TestResult struct {
	Test     string `json:"test"`     // 测试名称
	Input    string `json:"input"`    // 测试输入
	Expected string `json:"expected"` // 期望结果
	Actual   string `json:"actual"`   // 实际结果
	Status   string `json:"status"`   // 测试状态（✓通过/✗失败/pending待测试）
}

// Task 任务（对应 task_plan.md）
type Task struct {
	ID           string        `json:"id"`                      // 任务唯一标识符
	UserID       string        `json:"user_id"`                 // 用户ID
	SessionID    string        `json:"session_id"`              // 会话ID
	Goal         string        `json:"goal"`                    // 任务目标
	CurrentPhase string        `json:"current_phase"`           // 当前执行阶段ID
	Phases       []TaskPhase   `json:"phases"`                  // 任务阶段列表
	KeyQuestions []string      `json:"key_questions,omitempty"` // 关键问题列表
	Decisions    []Decision    `json:"decisions,omitempty"`     // 决策记录列表
	Errors       []ErrorRecord `json:"errors,omitempty"`        // 错误记录列表
	Status       TaskStatus    `json:"status"`                  // 任务状态（pending/in_progress/completed/failed/cancelled）
	CreatedAt    time.Time     `json:"created_at"`              // 创建时间
	UpdatedAt    time.Time     `json:"updated_at"`              // 更新时间
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`  // 完成时间

	// 上下文管理元数据
	ToolCallCount int  `json:"tool_call_count"` // 工具调用计数，用于决定何时重读计划（Manus的10次规则）
	NeedsReread   bool `json:"needs_reread"`    // 标记是否需要重读计划
}

// TaskFindings 任务发现（对应 findings.md）
type TaskFindings struct {
	TaskID       string    `json:"task_id"`                // 关联的任务ID
	Requirements []string  `json:"requirements,omitempty"` // 需求列表
	Findings     []Finding `json:"findings,omitempty"`     // 发现记录列表
	Resources    []string  `json:"resources,omitempty"`    // 资源列表（文件、URL等）
	UpdatedAt    time.Time `json:"updated_at"`             // 更新时间
}

// TaskProgress 任务进度（对应 progress.md）
type TaskProgress struct {
	TaskID      string          `json:"task_id"`                // 关联的任务ID
	SessionDate string          `json:"session_date"`           // 会话日期（格式：YYYY-MM-DD）
	Entries     []ProgressEntry `json:"entries,omitempty"`      // 进度条目列表
	TestResults []TestResult    `json:"test_results,omitempty"` // 测试结果列表
	ErrorLog    []ErrorRecord   `json:"error_log,omitempty"`    // 错误日志列表
	UpdatedAt   time.Time       `json:"updated_at"`             // 更新时间
}

// TaskContext 任务上下文（包含任务、发现和进度）
type TaskContext struct {
	Task     *Task         `json:"task"`     // 任务信息
	Findings *TaskFindings `json:"findings"` // 任务发现
	Progress *TaskProgress `json:"progress"` // 任务进度
}

// ContextCompression 上下文压缩配置
type ContextCompression struct {
	MaxToolResultsInContext int  `json:"max_tool_results_in_context"` // 上下文中保留的最大工具结果数（用于KV缓存优化）
	CompressOlderResults    bool `json:"compress_older_results"`      // 是否压缩较旧的结果
	KeepReferencesOnly      bool `json:"keep_references_only"`        // 只保留引用（文件路径、URL等），不保留完整内容
}

// TaskManagerConfig 任务管理器配置
type TaskManagerConfig struct {
	// 存储配置
	StorageType    StorageType `json:"storage_type"`     // 存储类型（file文件/db数据库/hybrid混合）
	StoragePath    string      `json:"storage_path"`     // 文件存储路径
	EnableFileSync bool        `json:"enable_file_sync"` // 混合模式下是否同步到文件系统

	// 行为配置
	RereadThreshold      int                `json:"reread_threshold"`     // 重读计划的工具调用阈值（Manus的10次规则，默认10）
	TwoActionRuleEnabled bool               `json:"two_action_rule"`      // 是否启用2动作规则（每2次视图操作后保存发现）
	Compression          ContextCompression `json:"compression"`          // 上下文压缩配置（用于KV缓存优化）
	MaxRetries           int                `json:"max_retries"`          // 最大重试次数（3次打击规则，默认3）
	EnableAutoPlanning   bool               `json:"enable_auto_planning"` // 是否启用LLM自动规划
}

// DefaultTaskManagerConfig 返回默认配置
func DefaultTaskManagerConfig() *TaskManagerConfig {
	return &TaskManagerConfig{
		StorageType:          StorageTypeFile, // 默认使用文件存储（符合 Manus 理念）
		StoragePath:          constant.DefaultTaskStoragePath,
		EnableFileSync:       true,
		RereadThreshold:      constant.DefaultRereadThreshold,
		TwoActionRuleEnabled: true,
		Compression: ContextCompression{
			MaxToolResultsInContext: constant.DefaultMaxToolResultsInContext,
			CompressOlderResults:    true,
			KeepReferencesOnly:      false,
		},
		MaxRetries:         constant.DefaultMaxRetries,
		EnableAutoPlanning: true,
	}
}

// ToolCall 工具调用记录
type ToolCall struct {
	ID         string                 `json:"id"`               // 工具调用唯一标识符
	Name       string                 `json:"name"`             // 工具名称
	Args       map[string]interface{} `json:"args,omitempty"`   // 工具调用参数
	Result     string                 `json:"result,omitempty"` // 工具调用结果
	Error      string                 `json:"error,omitempty"`  // 工具调用错误（如果有）
	Timestamp  time.Time              `json:"timestamp"`        // 调用时间
	Compressed bool                   `json:"compressed"`       // 是否已压缩（用于上下文压缩）
}

// PlanRequest 规划请求
type PlanRequest struct {
	UserID      string   `json:"user_id" binding:"required"`    // 用户ID（必填）
	SessionID   string   `json:"session_id" binding:"required"` // 会话ID（必填）
	Goal        string   `json:"goal" binding:"required"`       // 任务目标（必填）
	Context     string   `json:"context,omitempty"`             // 额外上下文信息
	Constraints []string `json:"constraints,omitempty"`         // 约束条件列表
	Preferences []string `json:"preferences,omitempty"`         // 偏好设置列表
}

// PlanResponse 规划响应
type PlanResponse struct {
	TaskID   string      `json:"task_id"`            // 创建的任务ID
	Goal     string      `json:"goal"`               // 任务目标
	Phases   []TaskPhase `json:"phases"`             // 规划的阶段列表
	Estimate string      `json:"estimate,omitempty"` // 预估完成时间
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	TaskID  string `json:"task_id" binding:"required"` // 任务ID（必填）
	PhaseID string `json:"phase_id,omitempty"`         // 可选，指定执行特定阶段
	StepID  string `json:"step_id,omitempty"`          // 可选，指定执行特定步骤
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	TaskID       string     `json:"task_id"`               // 任务ID
	CurrentPhase string     `json:"current_phase"`         // 当前执行阶段ID
	Status       TaskStatus `json:"status"`                // 任务状态
	Message      string     `json:"message"`               // 执行结果消息
	NextAction   string     `json:"next_action,omitempty"` // 建议的下一步操作
}

// TaskSummary 任务摘要（用于上下文压缩）
type TaskSummary struct {
	TaskID          string   `json:"task_id"`          // 任务ID
	Goal            string   `json:"goal"`             // 任务目标
	CurrentPhase    string   `json:"current_phase"`    // 当前阶段ID
	CompletedPhases []string `json:"completed_phases"` // 已完成的阶段ID列表
	KeyDecisions    []string `json:"key_decisions"`    // 关键决策摘要列表
	RecentErrors    []string `json:"recent_errors"`    // 最近的错误摘要列表
	Summary         string   `json:"summary"`          // 任务整体摘要
}
