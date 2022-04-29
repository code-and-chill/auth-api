package mysql

import (
	"context"
	"github.com/code-and-chill/auth-api/pkg/transaction"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type transactionProvider struct {
	db MySQL
}

// NewTransactionProvider instantiates a new transaction provider.
func NewTransactionProvider(db MySQL) (transaction.Provider, error) {
	return &transactionProvider{db}, nil
}

// WithTransaction wraps multiple unit of works with transaction.
func (t *transactionProvider) WithTransaction(ctx context.Context, unitOfWorks ...transaction.UnitOfWork) (interface{}, error) {
	result, err := t.db.WithTransaction(func(tx *sqlx.Tx, ch chan Result) {
		txRes := Result{Data: nil, Error: nil}
		ctx = context.WithValue(ctx, transaction.Context, tx)
		var resultData []interface{}
		for _, unitOfWork := range unitOfWorks {
			uowData, uowErr := unitOfWork.Execute(ctx, unitOfWork.Data)
			if uowErr != nil {
				txRes.Error = uowErr
				ch <- txRes
				return
			}
			resultData = append(resultData, uowData)
		}
		txRes.Data = resultData
		ch <- txRes
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	if result.Error != nil {
		return nil, errors.WithStack(result.Error)
	}
	return result.Data, nil
}
