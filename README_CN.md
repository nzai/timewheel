# Go分层时间轮实现

基于分层设计的时间轮算法实现，提供高效、精准的定时任务管理，支持长时间跨度任务处理。

## 特性

- 分层时间轮设计（支持秒/分钟/小时级任务）
- 线程安全实现
- 自动处理长周期任务降级
- 支持任务增删改查
- 过期回调机制
- 精确到基础时间间隔的触发精度

## 安装

```go
import "github.com/nzai/timewheel"  // 根据实际路径调整
```

## 快速开始

```go
import (
	"fmt"
	"time"

	"github.com/nzai/timewheel"
)

// 初始化时间轮（基础间隔1秒，每层60槽）
tw := timewheel.NewTimeWheel(time.Second, 60, func(key string, value interface{}) {
    fmt.Printf("[EXPIRED] Key:%s Value:%v\n", key, value)
})

// 添加30秒后过期的任务
tw.Set("key1", "data1", 30*time.Second)

// 修改任务过期时间（延长/缩短）
tw.Move("key1", 15*time.Minute)

// 删除任务
tw.Delete("key1")

// 清空所有key
tw.FlushAll()

// 停止时间轮
tw.Stop()
