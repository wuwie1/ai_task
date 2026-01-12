package tools

import (
	"errors"
	"github.com/gin-gonic/gin"
	"strings"
)

func ParseGinGetParams(ctx *gin.Context) (res map[string]string, err error) {
	return UriToMap(ctx.Request.URL.RawQuery)
}

func UriToMap(str string) (res map[string]string, err error) {
	res = make(map[string]string)
	if len(str) < 1 { // 空字符串
		err = errors.New("string is empty")
		return
	}
	if str[0:1] == "?" {
		str = str[1:]
	}
	pars := strings.Split(str, "&")
	for _, par := range pars {
		temp := strings.Split(par, "=")
		res[temp[0]] = temp[1]
	}
	return
}
