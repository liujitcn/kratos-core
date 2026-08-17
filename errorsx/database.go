package errorsx

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// IsDuplicateKey 判断是否命中了唯一键冲突。
func IsDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
