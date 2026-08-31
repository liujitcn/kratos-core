package audit

import (
	"context"
	"sync"
)

// Emitter 接收 Core 产生的访问审计事件。
type Emitter interface {
	Emit(context.Context, Event) error
}

var emitterState struct {
	sync.RWMutex
	emitter Emitter
}

// SetEmitter 设置当前进程的审计事件接收器。
func SetEmitter(value Emitter) {
	emitterState.Lock()
	emitterState.emitter = value
	emitterState.Unlock()
}

// Emit 将事件交给当前接收器；未配置接收器时静默忽略。
func Emit(ctx context.Context, event Event) error {
	emitterState.RLock()
	value := emitterState.emitter
	emitterState.RUnlock()
	if value == nil {
		return nil
	}
	return value.Emit(ctx, event)
}
