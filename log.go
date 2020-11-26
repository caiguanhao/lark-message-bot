package main

import (
	"os"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("lark")

func initLogger(verbosity string) {
	var lvl logging.Level
	switch verbosity {
	case "debug":
		lvl = logging.DEBUG
	case "notice":
		lvl = logging.NOTICE
	case "warning":
		lvl = logging.WARNING
	case "error":
		lvl = logging.ERROR
	case "critical":
		lvl = logging.CRITICAL
	default:
		lvl = logging.INFO
	}
	format := logging.MustStringFormatter("%{level: 7s} %{time:2006-01-02 15:04:05} %{message}")
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	formatter := logging.NewBackendFormatter(backend, format)
	level := logging.AddModuleLevel(formatter)
	level.SetLevel(lvl, "")
	logging.SetBackend(level)
}
