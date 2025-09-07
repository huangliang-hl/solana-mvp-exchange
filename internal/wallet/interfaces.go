package wallet

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// DBInterface 数据库接口
type DBInterface interface {
	WithTx(ctx context.Context, fn func(pgx.Tx) error) error
}
