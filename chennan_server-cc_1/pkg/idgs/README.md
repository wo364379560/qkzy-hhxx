# igs — ID Generation System

基于 Snowflake 算法的轻量唯一 ID 生成器，输出 `int64`，适用于数据库主键（BIGINT）。

## 位布局

```
┌──────────────────────────┬────────────┬────────────┐
│       timestamp          │    node    │    step    │
│         41-bit           │   10-bit   │   12-bit   │
├──────────────────────────┼────────────┼────────────┤
│  bit 63              22  │  21     12 │  11      0 │
└──────────────────────────┴────────────┴────────────┘
```

- **Epoch**：`2026-01-01 00:00:00 UTC`（`1767225600000 ms`），寿命 ~69.7 年
- **node**：0 ~ 1023，节点标识
- **step**：0 ~ 4095，同一毫秒内自增，耗尽忙等待下一毫秒

## 使用

```go
import "github.com/ixayda/ix-Go/pkg/igs"

n, err := igs.NewNode(1)
if err != nil {
    log.Fatal(err)
}
id := n.Generate() // igs.ID，可 .Int64() / .String()
```

## 错误

| 错误 | 触发条件 |
|------|---------|
| `ErrInvalidNode` | node 超出 [0, 1023] |
