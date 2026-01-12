package model

import "time"

// ========== 任务查询条件 ==========

// TaskQueryCondition 任务查询条件
type TaskQueryCondition struct {
	UserID    *string    `json:"user_id"`
	SessionID *string    `json:"session_id"`
	Status    *string    `json:"status"`
	Keyword   *string    `json:"keyword"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Offset    int        `json:"offset"`
	Limit     int        `json:"limit"`
	OrderBy   string     `json:"order_by"`
	OrderDesc bool       `json:"order_desc"`
}

// TaskListCondition 任务列表条件
type TaskListCondition struct {
	UserID    *string `json:"user_id"`
	SessionID *string `json:"session_id"`
}

// UpsertTaskCondition 创建/更新任务条件
type UpsertTaskCondition struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	SessionID     string     `json:"session_id"`
	Goal          string     `json:"goal"`
	CurrentPhase  *string    `json:"current_phase"`
	PhasesJSON    *string    `json:"phases_json"`
	QuestionsJSON *string    `json:"questions_json"`
	DecisionsJSON *string    `json:"decisions_json"`
	ErrorsJSON    *string    `json:"errors_json"`
	Status        *string    `json:"status"`
	ToolCallCount *int       `json:"tool_call_count"`
	NeedsReread   *bool      `json:"needs_reread"`
	CompletedAt   *time.Time `json:"completed_at"`
}

// ========== 任务发现查询条件 ==========

// UpsertTaskFindingsCondition 创建/更新任务发现条件
type UpsertTaskFindingsCondition struct {
	TaskID           string  `json:"task_id"`
	RequirementsJSON *string `json:"requirements_json"`
	FindingsJSON     *string `json:"findings_json"`
	ResourcesJSON    *string `json:"resources_json"`
}

// ========== 任务进度查询条件 ==========

// UpsertTaskProgressCondition 创建/更新任务进度条件
type UpsertTaskProgressCondition struct {
	TaskID          string  `json:"task_id"`
	SessionDate     *string `json:"session_date"`
	EntriesJSON     *string `json:"entries_json"`
	TestResultsJSON *string `json:"test_results_json"`
	ErrorLogJSON    *string `json:"error_log_json"`
}

// ========== 任务统计 ==========

// TaskStats 任务统计
type TaskStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}
