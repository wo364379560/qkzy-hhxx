package ix

import (
	"fmt"
)

var Prefix string
var Services string
var MicroService string
var ServicesVersion string

var MicroServiceName string = fmt.Sprintf("%s(%s)", Services, MicroService)
