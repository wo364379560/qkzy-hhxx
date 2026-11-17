package setup

import (
	"github.com/ixayda/ix-Go/internal/pkg/app/service"
	"github.com/ixayda/ix-Go/pkg/ix"
	"github.com/ixayda/ix-Go/pkg/logger"
)

func SystemBase() {
	app()
	appMode()
	appTimeZone()
}

func app() {
	logger.Infof("micro service: %s", ix.MicroService)
}

func appMode() {
	logger.Infof("System Running Mode: %s", service.MODE)
}

func appTimeZone() {
	logger.Infof("System Running TimeZone: %s", service.TimeZone)
}
