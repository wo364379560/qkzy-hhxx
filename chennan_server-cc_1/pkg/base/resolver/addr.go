package resolver

import (
	"strings"
)

func Addr(a, p string) string {
	var addr []string

	if a != "" {
		addr = append(addr, a)
	} else {
		addr = append(addr, "localhost")
	}

	if p != "" {
		addr = append(addr, p)
	} else {
		addr = append(addr, "8080")
	}

	return strings.Join(addr, ":")
}
