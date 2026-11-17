package idgs

import "errors"

const (
	// Epoch 2026-01-01 00:00:00 UTC（毫秒）
	Epoch int64 = 1767225600000

	nodeBits = 10 // node 字段位数，支持最多 1024 个节点
	stepBits = 12 // step 字段位数，每毫秒最多生成 4096 个 ID

	nodeMask = -1 ^ (-1 << nodeBits) // node 最大值掩码：1023
	stepMask = -1 ^ (-1 << stepBits) // step 最大值掩码：4095

	nodeShift      = stepBits            // node 字段左移位数
	timestampShift = nodeBits + stepBits // timestamp 字段左移位数
)

var ErrInvalidNode = errors.New("igs: node must be between 0 and 1023")
