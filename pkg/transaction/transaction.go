package transaction

import "context"

// UnitOfWork represents unit of work.
type UnitOfWork struct {
	Execute func(context.Context, interface{}) (interface{}, error)
	Data    interface{}
}

type transactionContext string

// Context TransactionContext represents transaction context.
const Context transactionContext = transactionContext("tx")

// Provider provides transaction context, commit and rollback.
type Provider interface {
	WithTransaction(context.Context, ...UnitOfWork) (interface{}, error)
}
