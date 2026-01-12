package tools

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func ErrorWithPrintContext(closeFunc func() error, format string, args ...interface{}) {
	if err := closeFunc(); err != nil {
		context := fmt.Sprintf(format, args...)
		log.Printf("Error closing resource: %s, error: %v", context, err)
	}
}
