package setup

import (
	"os"

	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/pkg/logger"
)

func SystemCore() {
	options := config.Cfg.Get()

	logger.Infof("system log level: %s", options.LogLevel)

	systemPath(options.Logdir)
	logger.Infof("system log path: %s", options.Logdir)

	systemPath(options.Datadir)
	logger.Infof("system data path: %s", options.Datadir)
}

func systemPath(dir string) {
	if dir == "" {
		return
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			panic(err)
		}
	}
}
