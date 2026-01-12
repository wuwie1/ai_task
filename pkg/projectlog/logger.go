package projectlog

import (
	"ai_web/test/config"
	"github.com/sirupsen/logrus"
	"os"
)

func Init() {
	//logrus.SetFormatter(&JSONFormatter{PrettyPrint: true})
	logrus.SetFormatter(&JSONFormatter{})
	level := logrus.Level(config.GetInstance().GetInt(config.AppLogLevel))
	logrus.SetLevel(level)
	rc := config.GetInstance().GetBool(config.AppLogReportcaller)
	logrus.SetReportCaller(rc)
	logrus.SetOutput(os.Stdout)
}
