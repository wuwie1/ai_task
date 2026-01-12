package main

import (
	"ai_web/test/config"
	"ai_web/test/pkg/projectlog"
	"ai_web/test/router"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	defer func() {
		if serviceErr := recover(); serviceErr != nil {
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			log.Println("The service exits abnormally, error message:【", serviceErr, "】")
			log.Println("Stack info: ")
			fmt.Printf("==> %s\n", string(buf[:n]))

			// @todo 发送报警信息
			os.Exit(1)
		}
	}()

	projectlog.Init()

	go startServer()
	waitStop()
}

func startServer() {
	addr := config.GetInstance().GetString(config.AppHost)
	if err := http.ListenAndServe(addr, router.GetInstance()); err != nil {
		logrus.Errorf("Failed to ListenAndServer at %v, err = %v", addr, err)
		os.Exit(1)
	}
}

func waitStop() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	sig := <-sc
	log.Printf("exit: signal=<%d>.\n", sig)
	switch sig {
	case syscall.SIGTERM:
		log.Println("exit: bye :-).")
		os.Exit(0)
	default:
		log.Println("exit: bye :-(.")
		os.Exit(1)
	}
}
