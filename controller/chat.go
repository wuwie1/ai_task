package controller

import (
	"ai_web/test/model"
	"ai_web/test/service/factory"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Chat 聊天接口
func Chat(ctx *gin.Context) {
	var reqBody struct {
		model.ChatRequest
		model.MemoryContextOptionsRequest `json:",inline"` // inline 作用：将 MemoryContextOptionsRequest 嵌入到 ChatRequest 中
	}

	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := reqBody.ChatRequest
	memoryOptions := &reqBody.MemoryContextOptionsRequest

	// 如果记忆选项为空，设置为 nil（使用默认值）
	if memoryOptions.SessionMemoryLimit == nil &&
		memoryOptions.SemanticMemoryLimit == nil &&
		memoryOptions.SemanticThreshold == nil {
		memoryOptions = nil
	}

	res, err := factory.GetServiceFactory().NewChatService().Chat(ctx, &req, memoryOptions)
	if err != nil {
		log.Errorf("Chat error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, res)
}
