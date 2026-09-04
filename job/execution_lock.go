package job

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bsm/redislock"
	"github.com/go-kratos/kratos/v3/log"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/locker"
)

const (
	executionLockKeyPrefix = "kratos:job:"
	executionLockTTL       = 5 * time.Minute
)

var (
	// ErrExecutionLockNotObtained 表示本次任务没有取得执行锁。
	ErrExecutionLockNotObtained = errors.New("任务执行锁未获取")
)

// ExecutionLocker 为定时任务提供 Redis 锁和单机内存锁两种实现。
type ExecutionLocker struct {
	distributed locker.Locker
	cleanup     func()
	local       *memoryExecutionLocker
	mode        string
	closeOnce   sync.Once
}

// ExecutionLease 表示一次已经取得的任务执行租约。
type ExecutionLease struct {
	ctx       context.Context
	releaseFn func() error
	release   sync.Once
	err       error
}

// NewExecutionLocker 创建任务执行锁；Redis 不可用时自动降级为单机内存锁。
func NewExecutionLocker(cfg *configv1.Data_Redis) *ExecutionLocker {
	manager := newMemoryExecutionLockerManager()
	distributed, cleanup, err := newDistributedLocker(cfg)
	if err != nil || distributed == nil {
		if cleanup != nil {
			cleanup()
		}
		if err == nil {
			err = errors.New("Redis 分布式锁实例为空")
		}
		log.Warn("任务分布式锁初始化失败，降级为单机内存锁", "error", err)
		return manager
	}
	manager.distributed = distributed
	manager.cleanup = cleanup
	manager.mode = "redis"
	log.Info("任务分布式锁已启用")
	return manager
}

// newDistributedLocker 隔离外部锁适配器的初始化 panic，保证锁失败时可以降级启动。
func newDistributedLocker(cfg *configv1.Data_Redis) (distributed locker.Locker, cleanup func(), err error) {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			distributed = nil
			cleanup = nil
			err = fmt.Errorf("初始化 Redis 分布式锁异常: %v", panicValue)
		}
	}()
	return locker.NewLocker(cfg)
}

// NewMemoryExecutionLocker 创建只使用进程内内存锁的任务执行锁。
func NewMemoryExecutionLocker() *ExecutionLocker {
	return newMemoryExecutionLockerManager()
}

// Mode 返回当前任务执行锁模式，值为 redis 或 memory。
func (l *ExecutionLocker) Mode() string {
	if l == nil || l.mode == "" {
		return "memory"
	}
	return l.mode
}

// Acquire 尝试为指定任务取得执行租约。
func (l *ExecutionLocker) Acquire(ctx context.Context, key string) (*ExecutionLease, error) {
	if l == nil {
		return nil, errors.New("任务执行锁未初始化")
	}
	if key == "" {
		return nil, errors.New("任务执行锁 key 不能为空")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if l.distributed == nil {
		if l.local == nil {
			return nil, errors.New("任务内存锁未初始化")
		}
		return l.local.acquire(ctx, key)
	}

	lock, err := l.distributed.Lock(key, int64(executionLockTTL/time.Second), nil)
	if err != nil {
		if errors.Is(err, redislock.ErrNotObtained) {
			return nil, ErrExecutionLockNotObtained
		}
		return nil, fmt.Errorf("获取任务分布式锁失败: %w", err)
	}
	if lock == nil {
		return nil, errors.New("获取任务分布式锁返回空租约")
	}
	return newRedisExecutionLease(ctx, lock, executionLockTTL), nil
}

// Close 释放任务执行锁持有的 Redis 客户端资源。
func (l *ExecutionLocker) Close() {
	if l == nil {
		return
	}
	l.closeOnce.Do(func() {
		if l.cleanup != nil {
			l.cleanup()
		}
	})
}

// Context 返回租约绑定的执行上下文。
func (l *ExecutionLease) Context() context.Context {
	if l == nil || l.ctx == nil {
		return context.Background()
	}
	return l.ctx
}

// Release 释放执行租约；重复释放只执行一次。
func (l *ExecutionLease) Release() error {
	if l == nil {
		return nil
	}
	l.release.Do(func() {
		if l.releaseFn != nil {
			l.err = l.releaseFn()
		}
	})
	return l.err
}

// newMemoryExecutionLockerManager 创建内存锁管理器。
func newMemoryExecutionLockerManager() *ExecutionLocker {
	return &ExecutionLocker{
		local: newMemoryExecutionLocker(),
		mode:  "memory",
	}
}

// newRedisExecutionLease 创建带自动续租的 Redis 执行租约。
func newRedisExecutionLease(parent context.Context, lock *redislock.Lock, ttl time.Duration) *ExecutionLease {
	leaseContext, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	refreshInterval := ttl / 3
	if refreshInterval < time.Second {
		refreshInterval = time.Second
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-leaseContext.Done():
				return
			case <-ticker.C:
				if err := lock.Refresh(context.Background(), ttl, nil); err != nil {
					log.Warn("任务分布式锁续租失败", "key", lock.Key(), "error", err)
					cancel()
					return
				}
			}
		}
	}()
	return &ExecutionLease{
		ctx: leaseContext,
		releaseFn: func() error {
			cancel()
			<-done
			err := lock.Release(context.Background())
			if errors.Is(err, redislock.ErrLockNotHeld) {
				return nil
			}
			return err
		},
	}
}

// memoryExecutionLocker 管理同一进程内按 key 隔离的互斥锁。
type memoryExecutionLocker struct {
	mu    sync.Mutex
	locks map[string]*memoryExecutionLock
}

type memoryExecutionLock struct {
	sem  chan struct{}
	refs int
}

// newMemoryExecutionLocker 创建内存执行锁管理器。
func newMemoryExecutionLocker() *memoryExecutionLocker {
	return &memoryExecutionLocker{locks: make(map[string]*memoryExecutionLock)}
}

// acquire 尝试取得指定 key 的进程内锁。
func (l *memoryExecutionLocker) acquire(ctx context.Context, key string) (*ExecutionLease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	l.mu.Lock()
	lock, ok := l.locks[key]
	if !ok {
		lock = &memoryExecutionLock{sem: make(chan struct{}, 1)}
		l.locks[key] = lock
	}
	lock.refs++
	l.mu.Unlock()

	select {
	case lock.sem <- struct{}{}:
		return &ExecutionLease{
			ctx: ctx,
			releaseFn: func() error {
				<-lock.sem
				l.mu.Lock()
				lock.refs--
				if lock.refs == 0 && l.locks[key] == lock {
					delete(l.locks, key)
				}
				l.mu.Unlock()
				return nil
			},
		}, nil
	default:
		l.mu.Lock()
		lock.refs--
		if lock.refs == 0 && l.locks[key] == lock {
			delete(l.locks, key)
		}
		l.mu.Unlock()
		return nil, ErrExecutionLockNotObtained
	}
}

// executionLockKey 返回跨实例共享的任务锁 key。
func executionLockKey(jobID int64) string {
	return fmt.Sprintf("%s%d", executionLockKeyPrefix, jobID)
}
