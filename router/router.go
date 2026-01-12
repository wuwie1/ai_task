package router

import (
	"sync"

	"github.com/gin-gonic/gin"
)

var once sync.Once
var instance *gin.Engine

func init() {
	once.Do(func() {
		instance = gin.New()
		addBasicRouter(instance)
		addApiRouter(instance)
	})
}

func GetInstance() *gin.Engine {
	return instance
}
