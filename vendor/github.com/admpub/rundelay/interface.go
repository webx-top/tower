package rundelay

import (
	"time"
)

var _ RunDelayer[any] = (*RunDelay[any])(nil)

type RunDelayer[T any] interface {
	Init(delay time.Duration, f func(T) error)
	Run(T) bool
	Done() error // 阻塞获取执行结果
	Close() error
}
