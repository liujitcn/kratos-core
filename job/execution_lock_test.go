package job

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/transport/cron"
)

// TestMemoryExecutionLockerSerializes 验证同一进程内同一任务只能取得一个租约。
func TestMemoryExecutionLockerSerializes(t *testing.T) {
	manager := NewMemoryExecutionLocker()
	first, err := manager.Acquire(context.Background(), "test-job")
	if err != nil {
		t.Fatalf("acquire first lease: %v", err)
	}
	defer first.Release()

	_, err = manager.Acquire(context.Background(), "test-job")
	if !errors.Is(err, ErrExecutionLockNotObtained) {
		t.Fatalf("expected lock contention, got %v", err)
	}
	if err = first.Release(); err != nil {
		t.Fatalf("release first lease: %v", err)
	}

	second, err := manager.Acquire(context.Background(), "test-job")
	if err != nil {
		t.Fatalf("acquire second lease after release: %v", err)
	}
	if err = second.Release(); err != nil {
		t.Fatalf("release second lease: %v", err)
	}
}

// TestMemoryExecutionLockerSeparatesKeys 验证不同任务 key 可以并行取得租约。
func TestMemoryExecutionLockerSeparatesKeys(t *testing.T) {
	manager := NewMemoryExecutionLocker()
	first, err := manager.Acquire(context.Background(), "job-a")
	if err != nil {
		t.Fatalf("acquire first lease: %v", err)
	}
	defer first.Release()
	second, err := manager.Acquire(context.Background(), "job-b")
	if err != nil {
		t.Fatalf("acquire second lease: %v", err)
	}
	if err = second.Release(); err != nil {
		t.Fatalf("release second lease: %v", err)
	}
}

// TestExecutionLockerFallsBackToMemory 验证 Redis 初始化失败时不阻断应用启动。
func TestExecutionLockerFallsBackToMemory(t *testing.T) {
	manager := NewExecutionLocker(nil)
	if manager.Mode() != "memory" {
		t.Fatalf("expected memory mode, got %s", manager.Mode())
	}
	lease, err := manager.Acquire(context.Background(), "test-job")
	if err != nil {
		t.Fatalf("acquire memory lease: %v", err)
	}
	if err = lease.Release(); err != nil {
		t.Fatalf("release memory lease: %v", err)
	}
}

// TestExecutionLockerFallsBackOnInvalidRedisConfig 验证外部锁初始化异常时仍能降级启动。
func TestExecutionLockerFallsBackOnInvalidRedisConfig(t *testing.T) {
	manager := NewExecutionLocker(&configv1.Data_Redis{})
	if manager.Mode() != "memory" {
		t.Fatalf("expected memory mode, got %s", manager.Mode())
	}
}

// TestExecutionLeaseReleaseIsIdempotent 验证租约重复释放不会重复操作底层锁。
func TestExecutionLeaseReleaseIsIdempotent(t *testing.T) {
	released := 0
	lease := &ExecutionLease{releaseFn: func() error {
		released++
		return nil
	}}
	if err := lease.Release(); err != nil {
		t.Fatalf("first release: %v", err)
	}
	if err := lease.Release(); err != nil {
		t.Fatalf("second release: %v", err)
	}
	if released != 1 {
		t.Fatalf("expected one release, got %d", released)
	}
}

type blockingTaskExec struct {
	started chan struct{}
	release chan struct{}
}

// Exec 执行阻塞任务，供调度器锁竞争测试使用。
func (t *blockingTaskExec) Exec(context.Context, map[string]string) ([]string, error) {
	close(t.started)
	<-t.release
	return []string{"ok"}, nil
}

// TestSchedulerSkipsContendedScheduledExecution 验证调度执行抢不到锁时只跳过本次执行。
func TestSchedulerSkipsContendedScheduledExecution(t *testing.T) {
	manager := NewMemoryExecutionLocker()
	scheduler := &Scheduler{locker: manager}
	task := &blockingTaskExec{started: make(chan struct{}), release: make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		result <- scheduler.executeJob(context.Background(), 1001, nil, cron.TaskExec(task), true)
	}()
	select {
	case <-task.started:
	case <-time.After(time.Second):
		t.Fatal("first scheduled execution did not start")
	}
	if err := scheduler.executeJob(context.Background(), 1001, nil, task, true); err != nil {
		t.Fatalf("expected contended scheduled execution to be skipped, got %v", err)
	}
	close(task.release)
	if err := <-result; err != nil {
		t.Fatalf("first scheduled execution failed: %v", err)
	}
}

// TestSchedulerReportsContendedManualExecution 验证手工执行抢不到锁时返回锁竞争错误。
func TestSchedulerReportsContendedManualExecution(t *testing.T) {
	manager := NewMemoryExecutionLocker()
	scheduler := &Scheduler{locker: manager}
	first, err := manager.Acquire(context.Background(), executionLockKey(1001))
	if err != nil {
		t.Fatalf("acquire first lease: %v", err)
	}
	defer first.Release()
	if err = scheduler.executeJob(context.Background(), 1001, nil, &blockingTaskExec{}, false); !errors.Is(err, ErrExecutionLockNotObtained) {
		t.Fatalf("expected manual execution contention, got %v", err)
	}
}

// TestMemoryExecutionLockerConcurrentAccess 验证内存锁在并发竞争下不会泄漏 key 状态。
func TestMemoryExecutionLockerConcurrentAccess(t *testing.T) {
	manager := NewMemoryExecutionLocker()
	var waitGroup sync.WaitGroup
	for i := 0; i < 100; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			lease, err := manager.Acquire(context.Background(), "concurrent-job")
			if err != nil {
				if !errors.Is(err, ErrExecutionLockNotObtained) {
					t.Errorf("acquire concurrent lease: %v", err)
				}
				return
			}
			if err = lease.Release(); err != nil {
				t.Errorf("release concurrent lease: %v", err)
			}
		}()
	}
	waitGroup.Wait()
	lease, err := manager.Acquire(context.Background(), "concurrent-job")
	if err != nil {
		t.Fatalf("acquire after concurrent access: %v", err)
	}
	if err = lease.Release(); err != nil {
		t.Fatalf("release after concurrent access: %v", err)
	}
}
