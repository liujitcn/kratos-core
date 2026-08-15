package data

import (
	"context"
	"errors"

	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
)

// Data 定义当前数据源的原生 GORM 查询实现。
type Data struct {
	db *gorm.DB
}

// Transaction 定义绑定当前数据源的事务执行能力。
type Transaction interface {
	Transaction(context.Context, func(context.Context) error) error
}

type transactionContextKey struct{}

// NewData 初始化数据访问对象。
func NewData(databases map[string]*databaseGorm.Client) (*Data, error) {
	if len(databases) == 0 {
		return nil, errors.New("数据库客户端映射不能为空")
	}
	defaultClient, ok := databases[databaseGorm.DefaultClientName]
	if !ok || defaultClient == nil || defaultClient.DB == nil {
		return nil, errors.New("默认数据库客户端不存在")
	}
	client := defaultClient
	return &Data{db: client.DB}, nil
}

// DB 返回绑定当前上下文的 GORM 数据库对象。
func (d *Data) DB(ctx context.Context) *gorm.DB {
	if d == nil || d.db == nil {
		return nil
	}
	if ctx != nil {
		if tx, ok := ctx.Value(transactionContextKey{}).(*gorm.DB); ok && tx != nil {
			return tx.WithContext(ctx)
		}
	}
	if ctx == nil {
		return d.db
	}
	return d.db.WithContext(ctx)
}

// Transaction 在当前数据源上执行事务，并把事务连接传递给回调内的仓储操作。
func (d *Data) Transaction(ctx context.Context, fn func(context.Context) error) error {
	if d == nil || d.db == nil {
		return errors.New("数据库连接未初始化")
	}
	if fn == nil {
		return errors.New("事务回调不能为空")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db := d.DB(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		transactionContext := context.WithValue(ctx, transactionContextKey{}, tx)
		return fn(transactionContext)
	})
}
