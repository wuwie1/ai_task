package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Storage 任务存储接口
type Storage interface {
	// 任务操作
	SaveTask(task *Task) error
	LoadTask(taskID string) (*Task, error)
	DeleteTask(taskID string) error
	ListTasks(userID, sessionID string) ([]*Task, error)

	// 发现操作
	SaveFindings(findings *TaskFindings) error
	LoadFindings(taskID string) (*TaskFindings, error)

	// 进度操作
	SaveProgress(progress *TaskProgress) error
	LoadProgress(taskID string) (*TaskProgress, error)

	// 完整上下文操作
	SaveContext(ctx *TaskContext) error
	LoadContext(taskID string) (*TaskContext, error)
}

// FileStorage 基于文件的任务存储实现
// 遵循 Manus 原则：文件系统作为外部记忆
type FileStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileStorage 创建文件存储
func NewFileStorage(basePath string) (*FileStorage, error) {
	// 确保目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileStorage{
		basePath: basePath,
	}, nil
}

// getTaskDir 获取任务目录
func (fs *FileStorage) getTaskDir(taskID string) string {
	return filepath.Join(fs.basePath, taskID)
}

// ensureTaskDir 确保任务目录存在
func (fs *FileStorage) ensureTaskDir(taskID string) error {
	dir := fs.getTaskDir(taskID)
	return os.MkdirAll(dir, 0755)
}

// SaveTask 保存任务（对应 task_plan.md）
func (fs *FileStorage) SaveTask(task *Task) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.ensureTaskDir(task.ID); err != nil {
		return err
	}

	task.UpdatedAt = time.Now()

	// 保存 JSON 格式
	jsonPath := filepath.Join(fs.getTaskDir(task.ID), "task.json")
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	// 同时保存 Markdown 格式（人类可读）
	mdPath := filepath.Join(fs.getTaskDir(task.ID), "task_plan.md")
	mdContent := fs.taskToMarkdown(task)
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		log.Warnf("Failed to write task markdown: %v", err)
	}

	return nil
}

// LoadTask 加载任务
func (fs *FileStorage) LoadTask(taskID string) (*Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	jsonPath := filepath.Join(fs.getTaskDir(taskID), "task.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// DeleteTask 删除任务
func (fs *FileStorage) DeleteTask(taskID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	dir := fs.getTaskDir(taskID)
	return os.RemoveAll(dir)
}

// ListTasks 列出任务
func (fs *FileStorage) ListTasks(userID, sessionID string) ([]*Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var tasks []*Task

	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return tasks, nil
		}
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		task, err := fs.LoadTask(entry.Name())
		if err != nil {
			log.Warnf("Failed to load task %s: %v", entry.Name(), err)
			continue
		}

		if task == nil {
			continue
		}

		// 过滤条件
		if userID != "" && task.UserID != userID {
			continue
		}
		if sessionID != "" && task.SessionID != sessionID {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// SaveFindings 保存发现（对应 findings.md）
func (fs *FileStorage) SaveFindings(findings *TaskFindings) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.ensureTaskDir(findings.TaskID); err != nil {
		return err
	}

	findings.UpdatedAt = time.Now()

	// 保存 JSON 格式
	jsonPath := filepath.Join(fs.getTaskDir(findings.TaskID), "findings.json")
	data, err := json.MarshalIndent(findings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal findings: %w", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write findings file: %w", err)
	}

	// 同时保存 Markdown 格式
	mdPath := filepath.Join(fs.getTaskDir(findings.TaskID), "findings.md")
	mdContent := fs.findingsToMarkdown(findings)
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		log.Warnf("Failed to write findings markdown: %v", err)
	}

	return nil
}

// LoadFindings 加载发现
func (fs *FileStorage) LoadFindings(taskID string) (*TaskFindings, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	jsonPath := filepath.Join(fs.getTaskDir(taskID), "findings.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read findings file: %w", err)
	}

	var findings TaskFindings
	if err := json.Unmarshal(data, &findings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal findings: %w", err)
	}

	return &findings, nil
}

// SaveProgress 保存进度（对应 progress.md）
func (fs *FileStorage) SaveProgress(progress *TaskProgress) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if err := fs.ensureTaskDir(progress.TaskID); err != nil {
		return err
	}

	progress.UpdatedAt = time.Now()

	// 保存 JSON 格式
	jsonPath := filepath.Join(fs.getTaskDir(progress.TaskID), "progress.json")
	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write progress file: %w", err)
	}

	// 同时保存 Markdown 格式
	mdPath := filepath.Join(fs.getTaskDir(progress.TaskID), "progress.md")
	mdContent := fs.progressToMarkdown(progress)
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		log.Warnf("Failed to write progress markdown: %v", err)
	}

	return nil
}

// LoadProgress 加载进度
func (fs *FileStorage) LoadProgress(taskID string) (*TaskProgress, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	jsonPath := filepath.Join(fs.getTaskDir(taskID), "progress.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read progress file: %w", err)
	}

	var progress TaskProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal progress: %w", err)
	}

	return &progress, nil
}

// SaveContext 保存完整上下文
func (fs *FileStorage) SaveContext(ctx *TaskContext) error {
	if ctx.Task != nil {
		if err := fs.SaveTask(ctx.Task); err != nil {
			return err
		}
	}
	if ctx.Findings != nil {
		if err := fs.SaveFindings(ctx.Findings); err != nil {
			return err
		}
	}
	if ctx.Progress != nil {
		if err := fs.SaveProgress(ctx.Progress); err != nil {
			return err
		}
	}
	return nil
}

// LoadContext 加载完整上下文
func (fs *FileStorage) LoadContext(taskID string) (*TaskContext, error) {
	task, err := fs.LoadTask(taskID)
	if err != nil {
		return nil, err
	}

	findings, err := fs.LoadFindings(taskID)
	if err != nil {
		return nil, err
	}

	progress, err := fs.LoadProgress(taskID)
	if err != nil {
		return nil, err
	}

	return &TaskContext{
		Task:     task,
		Findings: findings,
		Progress: progress,
	}, nil
}

// taskToMarkdown 将任务转换为 Markdown 格式
func (fs *FileStorage) taskToMarkdown(task *Task) string {
	md := fmt.Sprintf(`# Task Plan: %s

## Goal
%s

## Current Phase
%s

## Phases
`, task.ID, task.Goal, task.CurrentPhase)

	for _, phase := range task.Phases {
		md += fmt.Sprintf(`
### %s: %s
`, phase.ID, phase.Name)
		for _, step := range phase.Steps {
			checkbox := "[ ]"
			if step.Completed {
				checkbox = "[x]"
			}
			md += fmt.Sprintf("- %s %s\n", checkbox, step.Description)
		}
		md += fmt.Sprintf("- **Status:** %s\n", phase.Status)
	}

	if len(task.KeyQuestions) > 0 {
		md += "\n## Key Questions\n"
		for i, q := range task.KeyQuestions {
			md += fmt.Sprintf("%d. %s\n", i+1, q)
		}
	}

	if len(task.Decisions) > 0 {
		md += "\n## Decisions Made\n| Decision | Rationale |\n|----------|----------|\n"
		for _, d := range task.Decisions {
			md += fmt.Sprintf("| %s | %s |\n", d.Decision, d.Rationale)
		}
	}

	if len(task.Errors) > 0 {
		md += "\n## Errors Encountered\n| Error | Attempt | Resolution |\n|-------|---------|------------|\n"
		for _, e := range task.Errors {
			md += fmt.Sprintf("| %s | %d | %s |\n", e.Error, e.Attempt, e.Resolution)
		}
	}

	md += fmt.Sprintf(`
## Notes
- Update phase status as you progress: pending → in_progress → complete
- Re-read this plan before major decisions (attention manipulation)
- Log ALL errors - they help avoid repetition
- Task Status: %s
- Created: %s
- Updated: %s
`, task.Status, task.CreatedAt.Format(time.RFC3339), task.UpdatedAt.Format(time.RFC3339))

	return md
}

// findingsToMarkdown 将发现转换为 Markdown 格式
func (fs *FileStorage) findingsToMarkdown(findings *TaskFindings) string {
	md := fmt.Sprintf(`# Findings & Decisions

## Task ID
%s

## Requirements
`, findings.TaskID)

	for _, req := range findings.Requirements {
		md += fmt.Sprintf("- %s\n", req)
	}

	// 按类别分组发现
	researchFindings := []Finding{}
	technicalFindings := []Finding{}
	visualFindings := []Finding{}

	for _, f := range findings.Findings {
		switch f.Category {
		case "research":
			researchFindings = append(researchFindings, f)
		case "technical":
			technicalFindings = append(technicalFindings, f)
		case "visual":
			visualFindings = append(visualFindings, f)
		}
	}

	if len(researchFindings) > 0 {
		md += "\n## Research Findings\n"
		for _, f := range researchFindings {
			md += fmt.Sprintf("- %s", f.Content)
			if f.Source != "" {
				md += fmt.Sprintf(" (Source: %s)", f.Source)
			}
			md += "\n"
		}
	}

	if len(technicalFindings) > 0 {
		md += "\n## Technical Decisions\n"
		for _, f := range technicalFindings {
			md += fmt.Sprintf("- %s\n", f.Content)
		}
	}

	if len(visualFindings) > 0 {
		md += "\n## Visual/Browser Findings\n"
		for _, f := range visualFindings {
			md += fmt.Sprintf("- %s\n", f.Content)
		}
	}

	if len(findings.Resources) > 0 {
		md += "\n## Resources\n"
		for _, r := range findings.Resources {
			md += fmt.Sprintf("- %s\n", r)
		}
	}

	md += fmt.Sprintf(`
---
*Updated: %s*
*Update this file after every 2 view/browser/search operations*
`, findings.UpdatedAt.Format(time.RFC3339))

	return md
}

// progressToMarkdown 将进度转换为 Markdown 格式
func (fs *FileStorage) progressToMarkdown(progress *TaskProgress) string {
	md := fmt.Sprintf(`# Progress Log

## Task ID
%s

## Session: %s

`, progress.TaskID, progress.SessionDate)

	// 按阶段分组进度条目
	phaseEntries := make(map[string][]ProgressEntry)
	for _, entry := range progress.Entries {
		phaseEntries[entry.PhaseID] = append(phaseEntries[entry.PhaseID], entry)
	}

	for phaseID, entries := range phaseEntries {
		md += fmt.Sprintf("### %s\n", phaseID)
		md += "- Actions taken:\n"
		for _, entry := range entries {
			md += fmt.Sprintf("  - %s (%s)\n", entry.Action, entry.Timestamp.Format("15:04:05"))
			if len(entry.Files) > 0 {
				md += "    Files: "
				for i, f := range entry.Files {
					if i > 0 {
						md += ", "
					}
					md += f
				}
				md += "\n"
			}
		}
	}

	if len(progress.TestResults) > 0 {
		md += "\n## Test Results\n| Test | Input | Expected | Actual | Status |\n|------|-------|----------|--------|--------|\n"
		for _, t := range progress.TestResults {
			md += fmt.Sprintf("| %s | %s | %s | %s | %s |\n", t.Test, t.Input, t.Expected, t.Actual, t.Status)
		}
	}

	if len(progress.ErrorLog) > 0 {
		md += "\n## Error Log\n| Timestamp | Error | Attempt | Resolution |\n|-----------|-------|---------|------------|\n"
		for _, e := range progress.ErrorLog {
			md += fmt.Sprintf("| %s | %s | %d | %s |\n",
				e.Timestamp.Format("2006-01-02 15:04:05"), e.Error, e.Attempt, e.Resolution)
		}
	}

	md += fmt.Sprintf(`
---
*Updated: %s*
`, progress.UpdatedAt.Format(time.RFC3339))

	return md
}
