package entity

import "time"

// ========== 任务表 ==========

const (
	TableNameTask = "tasks"

	TaskFieldID            = "id"
	TaskFieldUserID        = "user_id"
	TaskFieldSessionID     = "session_id"
	TaskFieldGoal          = "goal"
	TaskFieldCurrentPhase  = "current_phase"
	TaskFieldPhasesJSON    = "phases_json"
	TaskFieldQuestionsJSON = "questions_json"
	TaskFieldDecisionsJSON = "decisions_json"
	TaskFieldErrorsJSON    = "errors_json"
	TaskFieldStatus        = "status"
	TaskFieldToolCallCount = "tool_call_count"
	TaskFieldNeedsReread   = "needs_reread"
	TaskFieldCreatedAt     = "created_at"
	TaskFieldUpdatedAt     = "updated_at"
	TaskFieldCompletedAt   = "completed_at"
)

// Task 任务数据库实体
type Task struct {
	ID            string     `xorm:"pk varchar(64) 'id'" json:"id"`
	UserID        string     `xorm:"varchar(64) index 'user_id'" json:"user_id"`
	SessionID     string     `xorm:"varchar(64) index 'session_id'" json:"session_id"`
	Goal          string     `xorm:"text 'goal'" json:"goal"`
	CurrentPhase  string     `xorm:"varchar(64) 'current_phase'" json:"current_phase"`
	PhasesJSON    string     `xorm:"text 'phases_json'" json:"phases_json"`
	QuestionsJSON string     `xorm:"text 'questions_json'" json:"questions_json"`
	DecisionsJSON string     `xorm:"text 'decisions_json'" json:"decisions_json"`
	ErrorsJSON    string     `xorm:"text 'errors_json'" json:"errors_json"`
	Status        string     `xorm:"varchar(32) index 'status'" json:"status"`
	ToolCallCount int        `xorm:"int 'tool_call_count'" json:"tool_call_count"`
	NeedsReread   bool       `xorm:"bool 'needs_reread'" json:"needs_reread"`
	CreatedAt     time.Time  `xorm:"created 'created_at'" json:"created_at"`
	UpdatedAt     time.Time  `xorm:"updated 'updated_at'" json:"updated_at"`
	CompletedAt   *time.Time `xorm:"'completed_at'" json:"completed_at"`
}

func (e *Task) TableName() string {
	return TableNameTask
}

// ========== 任务发现表 ==========

const (
	TableNameTaskFindings = "task_findings"

	TaskFindingsFieldID               = "id"
	TaskFindingsFieldTaskID           = "task_id"
	TaskFindingsFieldRequirementsJSON = "requirements_json"
	TaskFindingsFieldFindingsJSON     = "findings_json"
	TaskFindingsFieldResourcesJSON    = "resources_json"
	TaskFindingsFieldUpdatedAt        = "updated_at"
)

// TaskFindings 任务发现数据库实体
type TaskFindings struct {
	ID               int64     `xorm:"pk autoincr 'id'" json:"id"`
	TaskID           string    `xorm:"varchar(64) index 'task_id'" json:"task_id"`
	RequirementsJSON string    `xorm:"text 'requirements_json'" json:"requirements_json"`
	FindingsJSON     string    `xorm:"text 'findings_json'" json:"findings_json"`
	ResourcesJSON    string    `xorm:"text 'resources_json'" json:"resources_json"`
	UpdatedAt        time.Time `xorm:"updated 'updated_at'" json:"updated_at"`
}

func (e *TaskFindings) TableName() string {
	return TableNameTaskFindings
}

// ========== 任务进度表 ==========

const (
	TableNameTaskProgress = "task_progress"

	TaskProgressFieldID              = "id"
	TaskProgressFieldTaskID          = "task_id"
	TaskProgressFieldSessionDate     = "session_date"
	TaskProgressFieldEntriesJSON     = "entries_json"
	TaskProgressFieldTestResultsJSON = "test_results_json"
	TaskProgressFieldErrorLogJSON    = "error_log_json"
	TaskProgressFieldUpdatedAt       = "updated_at"
)

// TaskProgress 任务进度数据库实体
type TaskProgress struct {
	ID              int64     `xorm:"pk autoincr 'id'" json:"id"`
	TaskID          string    `xorm:"varchar(64) index 'task_id'" json:"task_id"`
	SessionDate     string    `xorm:"varchar(16) 'session_date'" json:"session_date"`
	EntriesJSON     string    `xorm:"text 'entries_json'" json:"entries_json"`
	TestResultsJSON string    `xorm:"text 'test_results_json'" json:"test_results_json"`
	ErrorLogJSON    string    `xorm:"text 'error_log_json'" json:"error_log_json"`
	UpdatedAt       time.Time `xorm:"updated 'updated_at'" json:"updated_at"`
}

func (e *TaskProgress) TableName() string {
	return TableNameTaskProgress
}
