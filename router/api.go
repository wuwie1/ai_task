package router

import (
	"ai_web/test/controller"

	"github.com/gin-gonic/gin"
)

func addApiRouter(engine *gin.Engine) {

	// 聊天相关 API
	api := engine.Group("/api/v1")
	{
		api.POST("/chat", controller.Chat)
	}
}
