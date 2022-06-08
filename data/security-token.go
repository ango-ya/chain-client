package data

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
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
