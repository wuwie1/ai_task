package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	RequestIDHeader    = "X-Request-ID"
	GinContextErrorKey = "Error"
	RequestIDInLogName = "request_id"
)

func Logger(ctx *gin.Context) {
	start := time.Now().UTC()
	path := ctx.Request.URL.Path
	var bodyBytes []byte
	if ctx.Request.Body != nil {
		bodyBytes, _ = io.ReadAll(ctx.Request.Body)
	}
	idr := io.NopCloser(bytes.NewBuffer(bodyBytes))
	request, err := readBody(idr)
	if err != nil {
		logrus.Errorf("read body bytes err:%v", err)
		return
	}
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	ip := ctx.ClientIP()

	ctx.Next()

	end := time.Now().UTC()
	latency := end.Sub(start)
	requestID, ok := ctx.Get(RequestIDHeader)
	if !ok {
		logrus.Infof("%s| %s| %s| %s |request: %s", ctx.Request.Method, latency, ip, path, request)
	} else {
		logrus.WithField(RequestIDInLogName, requestID).Infof("%s| %s| %s| %s |request: %s", ctx.Request.Method, latency, ip, path, request)
	}

}

func readBody(reader io.Reader) (string, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(reader)
	if err != nil {
		return "", errors.WithStack(err)
	}
	s := ""
	//if buf.Len() > 1024*5 {
	//	s = fmt.Sprintf("request size too big,request len = %v", buf.Len())
	//} else {
	//	s = buf.String()
	//}
	s = buf.String()
	return s, nil
}
