package idgs

import (
	"sync"
	"time"
)

// Node 是线程安全的 Snowflake ID 生成节点。
// 时间戳使用单调时钟，免疫 NTP 校正。
type Node struct {
	mu       sync.Mutex
	node     int64
	step     int64
	lastTime int64
	epoch    time.Time // 携带单调读数的 epoch 基准
}

// NewNode 创建一个 Node，node 范围 [0, 1023]。
func NewNode(node int64) (*Node, error) {
	if node < 0 || node > nodeMask {
		return nil, ErrInvalidNode
	}
	// 将 epoch 挂载到 curTime 的单调读数上（bwmarrin 技巧）：
	// time.Since(n.epoch) 走单调时钟路径，NTP 校正对其无效。
	cur := time.Now()
	epoch := cur.Add(time.UnixMilli(Epoch).Sub(cur))
	return &Node{node: node, epoch: epoch}, nil
}

// Generate 生成一个唯一 ID。
func (n *Node) Generate() ID {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := n.currentMs()

	if now == n.lastTime {
		n.step = (n.step + 1) & stepMask
		if n.step == 0 {
			// step 耗尽，忙等待直到时钟推进到下一毫秒
			for now <= n.lastTime {
				now = n.currentMs()
			}
		}
	} else {
		n.step = 0
	}

	n.lastTime = now
	return ID((now << timestampShift) | (n.node << nodeShift) | n.step)
}

// currentMs 返回自 epoch 起经过的毫秒数。
// 使用单调时钟，免疫 NTP 校正与时钟回拨。
func (n *Node) currentMs() int64 {
	return time.Since(n.epoch).Milliseconds()
}
