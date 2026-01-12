package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ai_task/entity"
	"ai_task/model"
	"ai_task/repository/factory"

	log "github.com/sirupsen/logrus"
)

// DBStorage 数据库存储实现
type DBStorage struct {
	factory        factory.Factory
	fileStorage    *FileStorage // 用于生成 Markdown 镜像
	mu             sync.RWMutex
	enableFileSync bool // 是否同步到文件
}

// 创建数据库存储
func NewDBStorage(f factory.Factory, filePath string, enableFileSync bool) (*DBStorage, error) {
	var fileStorage *FileStorage
	if enableFileSync && filePath != "" {
		var err error
		fileStorage, err = NewFileStorage(filePath)
		if err != nil {
			log.Warnf("Failed to create file storage for sync: %v", err)
		}
	}

	return &DBStorage{
		factory:        f,
		fileStorage:    fileStorage,
		enableFileSync: enableFileSync,
	}, nil
}

// SaveTask 保存任务
func (ds *DBStorage) SaveTask(task *Task) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	taskRepo, err := ds.factory.NewTaskRepository(session)
	if err != nil {
		return fmt.Errorf("failed to create task repository: %w", err)
	}

	req, err := ds.taskToCondition(task)
	if err != nil {
		return err
	}

	if err := taskRepo.Upsert(req); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// 同步到文件（异步，不阻塞主流程）
	if ds.enableFileSync && ds.fileStorage != nil {
		go func() {
			if err := ds.fileStorage.SaveTask(task); err != nil {
				log.Warnf("Failed to sync task to file: %v", err)
			}
		}()
	}

	return nil
}

// LoadTask 加载任务
func (ds *DBStorage) LoadTask(taskID string) (*Task, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	taskRepo, err := ds.factory.NewTaskRepository(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create task repository: %w", err)
	}

	record, err := taskRepo.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load task: %w", err)
	}

	if record == nil {
		return nil, nil
	}

	return ds.entityToTask(record)
}

// DeleteTask 删除任务
func (ds *DBStorage) DeleteTask(taskID string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	taskRepo, err := ds.factory.NewTaskRepository(session)
	if err != nil {
		return fmt.Errorf("failed to create task repository: %w", err)
	}

	if err := taskRepo.Delete(taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// 同步删除文件
	if ds.enableFileSync && ds.fileStorage != nil {
		go func() {
			if err := ds.fileStorage.DeleteTask(taskID); err != nil {
				log.Warnf("Failed to delete task files: %v", err)
			}
		}()
	}

	return nil
}

// ListTasks 列出任务
func (ds *DBStorage) ListTasks(userID, sessionID string) ([]*Task, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	taskRepo, err := ds.factory.NewTaskRepository(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create task repository: %w", err)
	}

	condition := &model.TaskListCondition{}
	if userID != "" {
		condition.UserID = &userID
	}
	if sessionID != "" {
		condition.SessionID = &sessionID
	}

	records, err := taskRepo.List(condition)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	tasks := make([]*Task, 0, len(records))
	for _, record := range records {
		task, err := ds.entityToTask(record)
		if err != nil {
			log.Warnf("Failed to convert task entity: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// SaveFindings 保存发现
func (ds *DBStorage) SaveFindings(findings *TaskFindings) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	findingsRepo, err := ds.factory.NewTaskFindingsRepository(session)
	if err != nil {
		return fmt.Errorf("failed to create findings repository: %w", err)
	}

	req, err := ds.findingsToCondition(findings)
	if err != nil {
		return err
	}

	if err := findingsRepo.Upsert(req); err != nil {
		return fmt.Errorf("failed to save findings: %w", err)
	}

	// 同步到文件
	if ds.enableFileSync && ds.fileStorage != nil {
		go func() {
			if err := ds.fileStorage.SaveFindings(findings); err != nil {
				log.Warnf("Failed to sync findings to file: %v", err)
			}
		}()
	}

	return nil
}

// LoadFindings 加载发现
func (ds *DBStorage) LoadFindings(taskID string) (*TaskFindings, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	findingsRepo, err := ds.factory.NewTaskFindingsRepository(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create findings repository: %w", err)
	}

	record, err := findingsRepo.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load findings: %w", err)
	}

	if record == nil {
		return nil, nil
	}

	return ds.entityToFindings(record)
}

// SaveProgress 保存进度
func (ds *DBStorage) SaveProgress(progress *TaskProgress) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	progressRepo, err := ds.factory.NewTaskProgressRepository(session)
	if err != nil {
		return fmt.Errorf("failed to create progress repository: %w", err)
	}

	req, err := ds.progressToCondition(progress)
	if err != nil {
		return err
	}

	if err := progressRepo.Upsert(req); err != nil {
		return fmt.Errorf("failed to save progress: %w", err)
	}

	// 同步到文件
	if ds.enableFileSync && ds.fileStorage != nil {
		go func() {
			if err := ds.fileStorage.SaveProgress(progress); err != nil {
				log.Warnf("Failed to sync progress to file: %v", err)
			}
		}()
	}

	return nil
}

// LoadProgress 加载进度
func (ds *DBStorage) LoadProgress(taskID string) (*TaskProgress, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	progressRepo, err := ds.factory.NewTaskProgressRepository(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create progress repository: %w", err)
	}

	record, err := progressRepo.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load progress: %w", err)
	}

	if record == nil {
		return nil, nil
	}

	return ds.entityToProgress(record)
}

// SaveContext 保存完整上下文
func (ds *DBStorage) SaveContext(ctx *TaskContext) error {
	if ctx.Task != nil {
		if err := ds.SaveTask(ctx.Task); err != nil {
			return err
		}
	}
	if ctx.Findings != nil {
		if err := ds.SaveFindings(ctx.Findings); err != nil {
			return err
		}
	}
	if ctx.Progress != nil {
		if err := ds.SaveProgress(ctx.Progress); err != nil {
			return err
		}
	}
	return nil
}

// LoadContext 加载完整上下文
func (ds *DBStorage) LoadContext(taskID string) (*TaskContext, error) {
	task, err := ds.LoadTask(taskID)
	if err != nil {
		return nil, err
	}

	findings, err := ds.LoadFindings(taskID)
	if err != nil {
		return nil, err
	}

	progress, err := ds.LoadProgress(taskID)
	if err != nil {
		return nil, err
	}

	return &TaskContext{
		Task:     task,
		Findings: findings,
		Progress: progress,
	}, nil
}

// QueryTasks 高级查询（数据库特有功能）
func (ds *DBStorage) QueryTasks(opts *TaskQueryOptions) ([]*Task, int64, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	taskRepo, err := ds.factory.NewTaskRepository(session)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create task repository: %w", err)
	}

	condition := &model.TaskQueryCondition{
		Offset:    opts.Offset,
		Limit:     opts.Limit,
		OrderBy:   opts.OrderBy,
		OrderDesc: opts.OrderDesc,
	}
	if opts.UserID != "" {
		condition.UserID = &opts.UserID
	}
	if opts.SessionID != "" {
		condition.SessionID = &opts.SessionID
	}
	if opts.Status != "" {
		condition.Status = &opts.Status
	}
	if opts.Keyword != "" {
		condition.Keyword = &opts.Keyword
	}
	if !opts.StartDate.IsZero() {
		condition.StartDate = &opts.StartDate
	}
	if !opts.EndDate.IsZero() {
		condition.EndDate = &opts.EndDate
	}

	records, total, err := taskRepo.Query(condition)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}

	tasks := make([]*Task, 0, len(records))
	for _, record := range records {
		task, err := ds.entityToTask(record)
		if err != nil {
			log.Warnf("Failed to convert task entity: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

// GetTaskStats 获取任务统计（数据库特有功能）
func (ds *DBStorage) GetTaskStats(userID string) (*TaskStats, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	ctx := context.Background()
	session := ds.factory.NewSession(ctx)
	defer func() { _ = session.Close() }()

	taskRepo, err := ds.factory.NewTaskRepository(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create task repository: %w", err)
	}

	stats, err := taskRepo.GetStats(userID)
	if err != nil {
		return nil, err
	}

	return &TaskStats{
		Total:      stats.Total,
		Pending:    stats.Pending,
		InProgress: stats.InProgress,
		Completed:  stats.Completed,
		Failed:     stats.Failed,
	}, nil
}

// ========== 转换方法 ==========

func (ds *DBStorage) taskToCondition(task *Task) (*model.UpsertTaskCondition, error) {
	phasesJSON, err := json.Marshal(task.Phases)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal phases: %w", err)
	}

	questionsJSON, err := json.Marshal(task.KeyQuestions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal questions: %w", err)
	}

	decisionsJSON, err := json.Marshal(task.Decisions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal decisions: %w", err)
	}

	errorsJSON, err := json.Marshal(task.Errors)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal errors: %w", err)
	}

	phasesStr := string(phasesJSON)
	questionsStr := string(questionsJSON)
	decisionsStr := string(decisionsJSON)
	errorsStr := string(errorsJSON)
	statusStr := string(task.Status)

	return &model.UpsertTaskCondition{
		ID:            task.ID,
		UserID:        task.UserID,
		SessionID:     task.SessionID,
		Goal:          task.Goal,
		CurrentPhase:  &task.CurrentPhase,
		PhasesJSON:    &phasesStr,
		QuestionsJSON: &questionsStr,
		DecisionsJSON: &decisionsStr,
		ErrorsJSON:    &errorsStr,
		Status:        &statusStr,
		ToolCallCount: &task.ToolCallCount,
		NeedsReread:   &task.NeedsReread,
		CompletedAt:   task.CompletedAt,
	}, nil
}

func (ds *DBStorage) entityToTask(record *entity.Task) (*Task, error) {
	var phases []TaskPhase
	if record.PhasesJSON != "" {
		if err := json.Unmarshal([]byte(record.PhasesJSON), &phases); err != nil {
			return nil, fmt.Errorf("failed to unmarshal phases: %w", err)
		}
	}

	var questions []string
	if record.QuestionsJSON != "" {
		if err := json.Unmarshal([]byte(record.QuestionsJSON), &questions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal questions: %w", err)
		}
	}

	var decisions []Decision
	if record.DecisionsJSON != "" {
		if err := json.Unmarshal([]byte(record.DecisionsJSON), &decisions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal decisions: %w", err)
		}
	}

	var errors []ErrorRecord
	if record.ErrorsJSON != "" {
		if err := json.Unmarshal([]byte(record.ErrorsJSON), &errors); err != nil {
			return nil, fmt.Errorf("failed to unmarshal errors: %w", err)
		}
	}

	return &Task{
		ID:            record.ID,
		UserID:        record.UserID,
		SessionID:     record.SessionID,
		Goal:          record.Goal,
		CurrentPhase:  record.CurrentPhase,
		Phases:        phases,
		KeyQuestions:  questions,
		Decisions:     decisions,
		Errors:        errors,
		Status:        TaskStatus(record.Status),
		ToolCallCount: record.ToolCallCount,
		NeedsReread:   record.NeedsReread,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
		CompletedAt:   record.CompletedAt,
	}, nil
}

func (ds *DBStorage) findingsToCondition(findings *TaskFindings) (*model.UpsertTaskFindingsCondition, error) {
	requirementsJSON, err := json.Marshal(findings.Requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal requirements: %w", err)
	}

	findingsJSON, err := json.Marshal(findings.Findings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal findings: %w", err)
	}

	resourcesJSON, err := json.Marshal(findings.Resources)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resources: %w", err)
	}

	requirementsStr := string(requirementsJSON)
	findingsStr := string(findingsJSON)
	resourcesStr := string(resourcesJSON)

	return &model.UpsertTaskFindingsCondition{
		TaskID:           findings.TaskID,
		RequirementsJSON: &requirementsStr,
		FindingsJSON:     &findingsStr,
		ResourcesJSON:    &resourcesStr,
	}, nil
}

func (ds *DBStorage) entityToFindings(record *entity.TaskFindings) (*TaskFindings, error) {
	var requirements []string
	if record.RequirementsJSON != "" {
		if err := json.Unmarshal([]byte(record.RequirementsJSON), &requirements); err != nil {
			return nil, fmt.Errorf("failed to unmarshal requirements: %w", err)
		}
	}

	var findings []Finding
	if record.FindingsJSON != "" {
		if err := json.Unmarshal([]byte(record.FindingsJSON), &findings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal findings: %w", err)
		}
	}

	var resources []string
	if record.ResourcesJSON != "" {
		if err := json.Unmarshal([]byte(record.ResourcesJSON), &resources); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
		}
	}

	return &TaskFindings{
		TaskID:       record.TaskID,
		Requirements: requirements,
		Findings:     findings,
		Resources:    resources,
		UpdatedAt:    record.UpdatedAt,
	}, nil
}

func (ds *DBStorage) progressToCondition(progress *TaskProgress) (*model.UpsertTaskProgressCondition, error) {
	entriesJSON, err := json.Marshal(progress.Entries)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entries: %w", err)
	}

	testResultsJSON, err := json.Marshal(progress.TestResults)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal test results: %w", err)
	}

	errorLogJSON, err := json.Marshal(progress.ErrorLog)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal error log: %w", err)
	}

	entriesStr := string(entriesJSON)
	testResultsStr := string(testResultsJSON)
	errorLogStr := string(errorLogJSON)

	return &model.UpsertTaskProgressCondition{
		TaskID:          progress.TaskID,
		SessionDate:     &progress.SessionDate,
		EntriesJSON:     &entriesStr,
		TestResultsJSON: &testResultsStr,
		ErrorLogJSON:    &errorLogStr,
	}, nil
}

func (ds *DBStorage) entityToProgress(record *entity.TaskProgress) (*TaskProgress, error) {
	var entries []ProgressEntry
	if record.EntriesJSON != "" {
		if err := json.Unmarshal([]byte(record.EntriesJSON), &entries); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entries: %w", err)
		}
	}

	var testResults []TestResult
	if record.TestResultsJSON != "" {
		if err := json.Unmarshal([]byte(record.TestResultsJSON), &testResults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test results: %w", err)
		}
	}

	var errorLog []ErrorRecord
	if record.ErrorLogJSON != "" {
		if err := json.Unmarshal([]byte(record.ErrorLogJSON), &errorLog); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error log: %w", err)
		}
	}

	return &TaskProgress{
		TaskID:      record.TaskID,
		SessionDate: record.SessionDate,
		Entries:     entries,
		TestResults: testResults,
		ErrorLog:    errorLog,
		UpdatedAt:   record.UpdatedAt,
	}, nil
}

// TaskQueryOptions 任务查询选项
type TaskQueryOptions struct {
	UserID    string
	SessionID string
	Status    string
	Keyword   string
	StartDate time.Time
	EndDate   time.Time
	Offset    int
	Limit     int
	OrderBy   string
	OrderDesc bool
}

// TaskStats 任务统计
type TaskStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}
