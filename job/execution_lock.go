package job

import (
	"context"
	"errors"
	"fmt"
	"time"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	kitlocker "github.com/liujitcn/kratos-kit/locker"
)

const (
	executionLockKeyPrefix = "kratos:job:"
	executionLockTTL       = 5 * time.Minute
)

var (
	// ErrExecutionLockNotObtained 表示本次任务没有取得执行锁。
	ErrExecutionLockNotObtained = kitlocker.ErrNotObtained
)

// ExecutionLocker 为定时任务封装统一锁能力。
type ExecutionLocker struct {
	locker kitlocker.Locker
}

// ExecutionLease 表示一次已经取得的任务执行租约。
type ExecutionLease struct {
	kitlocker.Lease
}

// NewExecutionLocker 根据 Redis 配置创建任务执行锁；未配置时使用进程内锁。
func NewExecutionLocker(cfg *configv1.Data_Redis) (*ExecutionLocker, error) {
	manager, err := kitlocker.NewLocker(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化任务执行锁失败: %w", err)
	}
	return &ExecutionLocker{locker: manager}, nil
}

// NewMemoryExecutionLocker 创建只使用进程内锁的任务执行锁。
func NewMemoryExecutionLocker() *ExecutionLocker {
	manager, err := kitlocker.NewLocker(nil)
	if err != nil {
		panic(err)
	}
	return &ExecutionLocker{locker: manager}
}

// Mode 返回当前任务执行锁模式。
func (l *ExecutionLocker) Mode() kitlocker.Mode {
	if l == nil || l.locker == nil {
		return kitlocker.ModeMemory
	}
	return l.locker.Mode()
}

// Acquire 尝试为指定任务取得执行租约。
func (l *ExecutionLocker) Acquire(ctx context.Context, key string) (*ExecutionLease, error) {
	if l == nil || l.locker == nil {
		return nil, errors.New("任务执行锁未初始化")
	}
	lease, err := l.locker.Acquire(ctx, key, executionLockTTL)
	if err != nil {
		return nil, err
	}
	return &ExecutionLease{Lease: lease}, nil
}

// Close 释放任务执行锁持有的资源。
func (l *ExecutionLocker) Close() {
	if l == nil || l.locker == nil {
		return
	}
	_ = l.locker.Close()
}

// executionLockKey 返回跨实例共享的任务锁 key。
func executionLockKey(jobID int64) string {
	return fmt.Sprintf("%s%d", executionLockKeyPrefix, jobID)
}
