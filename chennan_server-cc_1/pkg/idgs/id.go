package idgs

import (
	"fmt"
	"strconv"
	"sync"
)

// ID 是 Snowflake ID 的类型，底层为 int64。
type ID int64

// Int64 返回 int64 表示。
func (f ID) Int64() int64 {
	return int64(f)
}

// String 返回十进制字符串表示。
func (f ID) String() string {
	return strconv.FormatInt(int64(f), 10)
}

// Global 是全局 ID 生成器实例。
var Global = &globalInstance{
	once: sync.Once{},
}

type globalInstance struct {
	once      sync.Once
	userNode  *Node
	groupNode *Node
}

// Init 初始化全局 ID 生成器（线程安全，仅执行一次）。
// nodeUser 和 nodeGroup 范围均为 [0, 1023]。
func (g *globalInstance) Init(nodeUser, nodeGroup int64) error {
	var err error
	g.once.Do(func() {
		g.userNode, err = NewNode(nodeUser)
		if err != nil {
			err = fmt.Errorf("failed to create user node: %w", err)
			return
		}
		g.groupNode, err = NewNode(nodeGroup)
		if err != nil {
			err = fmt.Errorf("failed to create group node: %w", err)
			return
		}
	})
	return err
}

// UserNode 返回用户业务 ID 生成节点。
func (g *globalInstance) UserNode() *Node {
	return g.userNode
}

// GroupNode 返回群组业务 ID 生成节点。
func (g *globalInstance) GroupNode() *Node {
	return g.groupNode
}

// UserGenerate 生成用户业务 ID。
func (g *globalInstance) UserGenerate() ID {
	return g.userNode.Generate()
}

// GroupGenerate 生成群组业务 ID。
func (g *globalInstance) GroupGenerate() ID {
	return g.groupNode.Generate()
}
