package data

import "context"

// Transaction 提供宿主数据访问层的事务边界。
type Transaction interface {
	// Transaction 在事务中执行回调，并将事务上下文传递给宿主 Store。
	Transaction(context.Context, func(context.Context) error) error
}
