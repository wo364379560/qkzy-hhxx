/*
Copyright © 2026 iXayda https://iXayda.com
*/

package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/ixayda/ix-Go/internal/pkg/app/setup"
	"github.com/ixayda/ix-Go/internal/pkg/config"
	"github.com/ixayda/ix-Go/pkg/ix"
	"github.com/ixayda/ix-Go/pkg/logger"
)

func init() {
	_ = godotenv.Load()
	ix.ShowBanner()
}

type app struct {
	Usage   string
	Flags   func() (bool, error)
	Service func() error

	failed  int
	success int
}

func GetApp() *app {
	return &app{
		failed:  1,
		success: 0,
	}
}

func (app *app) Run() (int, error) {
	var err error
	var isBreak bool
	var isCapture bool

	if app.Service == nil {
		return app.failed, errors.New("exception: Service Instance NOT DEFINE")
	}

	if len(os.Args) > 1 {
		var argv = os.Args[1:]
		var argc = len(os.Args) - 1
		isBreak, isCapture, err = app.flags(argc, argv)
		if err != nil {
			return app.failed, err
		}
	}

	if isBreak {
		return app.success, err
	}

	if isCapture {
		isBreak, err = app.Flags()
		if err != nil {
			return app.failed, err
		}
		if isBreak {
			return app.success, err
		}
	}

	_ = godotenv.Load()

	err = config.Setup()
	if err != nil {
		return app.failed, err
	}

	logger.Init()

	setup.SystemBase()
	setup.SystemCore()

	err = app.Service()
	return app.success, err
}

func (app *app) flags(_ int, argv []string) (bool, bool, error) {
	var isBreak bool
	var isCapture bool

	switch argv[0] {
	case "-h", "--help", "help":
		isBreak = true
		fmt.Println(app.Usage)
	case "-v", "--version", "version":
		isBreak = true
		ix.Version.Print()
	default:
		isCapture = true
	}
	return isBreak, isCapture, nil
}

var App = GetApp()
