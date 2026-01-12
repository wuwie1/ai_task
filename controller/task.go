package controller

import (
	"ai_task/pkg/task"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	taskServiceOnce sync.Once
	taskService     *task.Service
)

// getTaskService 获取任务服务单例
func getTaskService() *task.Service {
	taskServiceOnce.Do(func() {
		var err error
		taskService, err = task.NewService(nil)
		if err != nil {
			log.Fatalf("Failed to create task service: %v", err)
		}
	})
	return taskService
}

// CreateTask 创建任务
// @Summary 创建新任务
// @Description 根据目标创建任务计划，支持 LLM 自动规划
// @Tags Task
// @Accept json
// @Produce json
// @Param request body task.PlanRequest true "任务请求"
// @Success 200 {object} task.PlanResponse
// @Router /api/v1/task [post]
func CreateTask(ctx *gin.Context) {
	var req task.PlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := getTaskService().CreateTask(ctx, &req)
	if err != nil {
		log.Errorf("CreateTask error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// GetTask 获取任务
// @Summary 获取任务详情
// @Description 根据任务ID获取任务详情
// @Tags Task
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} task.Task
// @Router /api/v1/task/{task_id} [get]
func GetTask(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	t, err := getTaskService().GetTask(ctx, taskID)
	if err != nil {
		log.Errorf("GetTask error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if t == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	ctx.JSON(http.StatusOK, t)
}

// GetTaskContext 获取任务上下文
// @Summary 获取完整任务上下文
// @Description 获取任务、发现和进度的完整上下文
// @Tags Task
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} task.TaskContext
// @Router /api/v1/task/{task_id}/context [get]
func GetTaskContext(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	taskCtx, err := getTaskService().GetTaskContext(ctx, taskID)
	if err != nil {
		log.Errorf("GetTaskContext error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if taskCtx == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	ctx.JSON(http.StatusOK, taskCtx)
}

// ListTasks 列出任务
// @Summary 列出任务
// @Description 根据用户ID和会话ID列出任务
// @Tags Task
// @Produce json
// @Param user_id query string false "用户ID"
// @Param session_id query string false "会话ID"
// @Success 200 {array} task.Task
// @Router /api/v1/tasks [get]
func ListTasks(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	sessionID := ctx.Query("session_id")

	tasks, err := getTaskService().ListTasks(ctx, userID, sessionID)
	if err != nil {
		log.Errorf("ListTasks error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

// ExecuteTask 执行任务
// @Summary 执行任务
// @Description 执行任务或指定阶段
// @Tags Task
// @Accept json
// @Produce json
// @Param request body task.ExecuteRequest true "执行请求"
// @Success 200 {object} task.ExecuteResponse
// @Router /api/v1/task/execute [post]
func ExecuteTask(ctx *gin.Context) {
	var req task.ExecuteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := getTaskService().ExecuteTask(ctx, &req)
	if err != nil {
		log.Errorf("ExecuteTask error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// UpdatePhaseRequest 更新阶段请求
type UpdatePhaseRequest struct {
	PhaseID string           `json:"phase_id" binding:"required"`
	Status  task.PhaseStatus `json:"status" binding:"required"`
}

// UpdatePhase 更新阶段状态
// @Summary 更新阶段状态
// @Description 更新任务的阶段状态
// @Tags Task
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param request body UpdatePhaseRequest true "更新请求"
// @Success 200 {object} gin.H
// @Router /api/v1/task/{task_id}/phase [put]
func UpdatePhase(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	var req UpdatePhaseRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := getTaskService().UpdatePhase(ctx, taskID, req.PhaseID, req.Status)
	if err != nil {
		log.Errorf("UpdatePhase error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "phase updated"})
}

// CompleteStepRequest 完成步骤请求
type CompleteStepRequest struct {
	PhaseID string `json:"phase_id" binding:"required"`
	StepID  string `json:"step_id" binding:"required"`
	Result  string `json:"result"`
}

// CompleteStep 完成步骤
// @Summary 完成步骤
// @Description 标记任务步骤为完成
// @Tags Task
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param request body CompleteStepRequest true "完成请求"
// @Success 200 {object} gin.H
// @Router /api/v1/task/{task_id}/step [put]
func CompleteStep(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	var req CompleteStepRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := getTaskService().CompleteStep(ctx, taskID, req.PhaseID, req.StepID, req.Result)
	if err != nil {
		log.Errorf("CompleteStep error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "step completed"})
}

// AddFindingRequest 添加发现请求
type AddFindingRequest struct {
	Category string `json:"category" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Source   string `json:"source"`
}

// AddFinding 添加发现
// @Summary 添加发现
// @Description 添加任务发现
// @Tags Task
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param request body AddFindingRequest true "发现请求"
// @Success 200 {object} gin.H
// @Router /api/v1/task/{task_id}/finding [post]
func AddFinding(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	var req AddFindingRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := getTaskService().AddFinding(ctx, taskID, req.Category, req.Content, req.Source)
	if err != nil {
		log.Errorf("AddFinding error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "finding added"})
}

// AddDecisionRequest 添加决策请求
type AddDecisionRequest struct {
	Decision  string `json:"decision" binding:"required"`
	Rationale string `json:"rationale" binding:"required"`
}

// AddDecision 添加决策
// @Summary 添加决策
// @Description 添加任务决策
// @Tags Task
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param request body AddDecisionRequest true "决策请求"
// @Success 200 {object} gin.H
// @Router /api/v1/task/{task_id}/decision [post]
func AddDecision(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	var req AddDecisionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := getTaskService().AddDecision(ctx, taskID, req.Decision, req.Rationale)
	if err != nil {
		log.Errorf("AddDecision error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "decision added"})
}

// RecordErrorRequest 记录错误请求
type RecordErrorRequest struct {
	Error      string `json:"error" binding:"required"`
	Attempt    int    `json:"attempt"`
	Resolution string `json:"resolution"`
}

// RecordError 记录错误
// @Summary 记录错误
// @Description 记录任务错误
// @Tags Task
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param request body RecordErrorRequest true "错误请求"
// @Success 200 {object} gin.H
// @Router /api/v1/task/{task_id}/error [post]
func RecordError(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	var req RecordErrorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := getTaskService().RecordError(ctx, taskID, req.Error, req.Attempt, req.Resolution)
	if err != nil {
		log.Errorf("RecordError error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "error recorded"})
}

// CheckCompletion 检查完成状态
// @Summary 检查完成状态
// @Description 检查任务是否完成
// @Tags Task
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} task.CompletionStatus
// @Router /api/v1/task/{task_id}/completion [get]
func CheckCompletion(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	status, err := getTaskService().CheckCompletion(ctx, taskID)
	if err != nil {
		log.Errorf("CheckCompletion error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, status)
}

// GetTaskSummary 获取任务摘要
// @Summary 获取任务摘要
// @Description 获取任务的压缩摘要
// @Tags Task
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} task.TaskSummary
// @Router /api/v1/task/{task_id}/summary [get]
func GetTaskSummary(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	summary, err := getTaskService().GetTaskSummary(ctx, taskID)
	if err != nil {
		log.Errorf("GetTaskSummary error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, summary)
}

// DeleteTask 删除任务
// @Summary 删除任务
// @Description 删除任务及其所有相关数据
// @Tags Task
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} gin.H
// @Router /api/v1/task/{task_id} [delete]
func DeleteTask(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	err := getTaskService().DeleteTask(ctx, taskID)
	if err != nil {
		log.Errorf("DeleteTask error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "task deleted"})
}

// StartSession 开始会话
// @Summary 开始会话
// @Description 开始一个新的任务会话
// @Tags Session
// @Accept json
// @Produce json
// @Param request body task.PlanRequest true "任务请求"
// @Success 200 {object} task.SessionInfo
// @Router /api/v1/session [post]
func StartSession(ctx *gin.Context) {
	var req task.PlanRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	info, err := getTaskService().StartSession(ctx, &req)
	if err != nil {
		log.Errorf("StartSession error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, info)
}

// ExecuteSession 执行会话
// @Summary 执行会话
// @Description 执行会话中的任务
// @Tags Session
// @Produce json
// @Param session_id path string true "会话ID"
// @Success 200 {object} task.ExecuteResponse
// @Router /api/v1/session/{session_id}/execute [post]
func ExecuteSession(ctx *gin.Context) {
	sessionID := ctx.Param("session_id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	resp, err := getTaskService().ExecuteSession(ctx, sessionID)
	if err != nil {
		log.Errorf("ExecuteSession error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// CheckSessionStop 检查会话停止
// @Summary 检查会话是否可以停止
// @Description 检查会话任务是否完成，可以停止
// @Tags Session
// @Produce json
// @Param session_id path string true "会话ID"
// @Success 200 {object} task.CompletionStatus
// @Router /api/v1/session/{session_id}/stop [get]
func CheckSessionStop(ctx *gin.Context) {
	sessionID := ctx.Param("session_id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	status, err := getTaskService().CheckSessionStop(ctx, sessionID)
	if err != nil {
		log.Errorf("CheckSessionStop error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, status)
}

// GetOptimizedContextRequest 获取优化上下文请求
type GetOptimizedContextRequest struct {
	ToolCalls []task.ToolCall `json:"tool_calls"`
}

// GetOptimizedContext 获取优化的上下文
// @Summary 获取优化的上下文
// @Description 获取经过压缩和优化的任务上下文
// @Tags Task
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param request body GetOptimizedContextRequest false "工具调用"
// @Success 200 {object} task.OptimizedContext
// @Router /api/v1/task/{task_id}/optimized-context [post]
func GetOptimizedContext(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	var req GetOptimizedContextRequest
	_ = ctx.ShouldBindJSON(&req)

	optimized, err := getTaskService().GetOptimizedContext(ctx, taskID, req.ToolCalls)
	if err != nil {
		log.Errorf("GetOptimizedContext error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, optimized)
}

// RecordViewActionRequest 记录视图动作请求
type RecordViewActionRequest struct {
	ActionType task.ActionType `json:"action_type" binding:"required"`
}

// RecordViewAction 记录视图动作
// @Summary 记录视图动作
// @Description 记录视图/浏览/搜索动作，用于2动作规则
// @Tags Task
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Param request body RecordViewActionRequest true "动作请求"
// @Success 200 {object} gin.H
// @Router /api/v1/task/{task_id}/view-action [post]
func RecordViewAction(ctx *gin.Context) {
	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	var req RecordViewActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	needsSave, err := getTaskService().RecordViewAction(ctx, taskID, req.ActionType)
	if err != nil {
		log.Errorf("RecordViewAction error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":     "action recorded",
		"needs_save":  needsSave,
		"action_rule": "2-action rule",
	})
}
