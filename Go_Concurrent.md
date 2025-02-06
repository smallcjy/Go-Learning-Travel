# Concurrent In GO
在并发编程中，对于共享变量的正确访问需要精确的控制以实现正确的同步，这是非常困难的。而在Go中，他将共享的值通过信道传递，实际上，多个独立执行的线程从不会主动共享。在任意给定的时间点上，只有一个goroutine能够访问该值，所以也就不存在并发访问共享数据带来的竞态问题。这个思想被通过一句口号来说明：

**Do not communicate by sharing memory; instead, share memory by communicating.**

## Gorouines / go协程

Goroutine是非常简单的模型，是和其他goroutine并发运行在同一个地址空间的函数。goroutine的切换仅仅涉及到栈的切换，栈是非常廉价的开销。在函数前面加个go，表示分配新的goroutine来运行这个函数。

```go
go func_name()
```

## Channels / 信道

## Context 控制上下文


# GO 竞态问题
## sync.Mutex
如果结构体内有竞态资源，需要用锁来保护
```go
type SafeCounter struct {
    v   map[string]int
    mux sync.Mutex
}

// 获取锁
func (c *SafeCounter) Inc(key string) {
    c.mux.Lock()
    defer c.mux.Unlock()
    c.v[key]++
}

```

