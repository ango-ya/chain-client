package data

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	ST_CONTROL_ROLE = "b6ce5d7b1abd7b8db19bd268a06356fe343d6a81aca7f86455289d12aecbdcda"
	ST_EDIT_ROLE    = "025c10ffb4b4f977a8899da54e53278bc52863e80645c6b1f1ee5085ab0069bc"
)

func ToWei(iamount interface{}, decimals int) (*big.Int, error) {
	var (
		amount = decimal.NewFromFloat(0)
		err    error
	)
	switch v := iamount.(type) {
	case string:
		if amount, err = decimal.NewFromString(v); err != nil {
			return nil, errors.Wrap(err, "failed to cast to wei")
		}
	case float64:
		amount = decimal.NewFromFloat(v)
	case int64:
		amount = decimal.NewFromFloat(float64(v))
	case decimal.Decimal:
		amount = v
	case *decimal.Decimal:
		amount = *v
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	result := amount.Mul(mul)

	wei := new(big.Int)
	wei.SetString(result.String(), 10)

	return wei, nil
}
