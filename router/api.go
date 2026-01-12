package router

import (
	"ai_task/controller"

	"github.com/gin-gonic/gin"
)

func addApiRouter(engine *gin.Engine) {

	// 聊天相关 API
	api := engine.Group("/api/v1")
	{
		api.POST("/chat", controller.Chat)

		// 任务管理 API
		// 任务 CRUD
		api.POST("/task", controller.CreateTask)
		api.GET("/task/:task_id", controller.GetTask)
		api.DELETE("/task/:task_id", controller.DeleteTask)
		api.GET("/tasks", controller.ListTasks)

		// 任务执行
		api.POST("/task/execute", controller.ExecuteTask)

		// 任务上下文
		api.GET("/task/:task_id/context", controller.GetTaskContext)
		api.GET("/task/:task_id/summary", controller.GetTaskSummary)
		api.POST("/task/:task_id/optimized-context", controller.GetOptimizedContext)

		// 阶段和步骤管理
		api.PUT("/task/:task_id/phase", controller.UpdatePhase)
		api.PUT("/task/:task_id/step", controller.CompleteStep)

		// 发现、决策和错误
		api.POST("/task/:task_id/finding", controller.AddFinding)
		api.POST("/task/:task_id/decision", controller.AddDecision)
		api.POST("/task/:task_id/error", controller.RecordError)

		// 完成检查
		api.GET("/task/:task_id/completion", controller.CheckCompletion)

		// 2动作规则
		api.POST("/task/:task_id/view-action", controller.RecordViewAction)

		// 会话管理
		api.POST("/session", controller.StartSession)
		api.POST("/session/:session_id/execute", controller.ExecuteSession)
		api.GET("/session/:session_id/stop", controller.CheckSessionStop)
	}
}
