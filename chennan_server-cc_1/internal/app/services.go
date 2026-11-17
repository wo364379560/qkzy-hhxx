/*
Copyright © 2026 iXayda https://iXayda.com
*/

package app

import "github.com/ixayda/ix-Go/internal/app/server"

type service string

const (
	Server service = "server"
)

// ServiceProvider 是每个服务必须实现的接口，新增服务只需实现此接口并在 registry 添加一条记录。
type ServiceProvider interface {
	Usage() string
	Flags() (bool, error)
	Execute() error
}

// registry 是服务名到 ServiceProvider 的映射，新增服务只改此处。
var registry = map[service]ServiceProvider{
	Server: server.Provider{},
}

func (s service) Usage() string {
	if def, ok := registry[s]; ok {
		return def.Usage()
	}
	return ""
}

func (s service) Flags() func() (bool, error) {
	if def, ok := registry[s]; ok {
		return def.Flags
	}
	return func() (bool, error) { return true, nil }
}

func (s service) Service() func() error {
	if def, ok := registry[s]; ok {
		return def.Execute
	}
	return nil
}
