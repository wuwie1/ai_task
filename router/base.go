package router

import (
	"github.com/gin-gonic/gin"
)

func addBasicRouter(engine *gin.Engine) {
	engine.Use(gin.Recovery())
}
