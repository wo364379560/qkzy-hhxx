/*
Copyright © 2026 iXayda https://iXayda.com
*/

package main

import (
	"fmt"
	"os"

	"github.com/ixayda/ix-Go/internal/app"
)

func main() {
	service := app.App
	service.Usage = app.Server.Usage()
	service.Flags = app.Server.Flags()
	service.Service = app.Server.Service()
	code, err := service.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}
