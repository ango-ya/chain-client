package data

import (
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	ST_CONTROL_ROLE = "b6ce5d7b1abd7b8db19bd268a06356fe343d6a81aca7f86455289d12aecbdcda"
	ST_EDIT_ROLE    = "025c10ffb4b4f977a8899da54e53278bc52863e80645c6b1f1ee5085ab0069bc"
)

func (r *SendETHRequest) Validate() error {
	if err := validateAddress(r.GetRecipient()); err != nil {
		return errors.Wrap(err, "invalid recipient")
	}
	if _, err := ToWei(r.GetAmount(), 18); err != nil {
		return errors.Wrapf(err, "invalid amount(=%v)", r.GetAmount())
	}
	return nil
}

func (r *BalanceOfETHRequest) Validate() error {
	if err := validateAddress(r.GetAccount()); err != nil {
		return errors.Wrap(err, "invalid account")
	}
	return nil
}

func (r *DeploySTRequest) Validate() error {
	if err := validateAddress(r.GetComplianceAddress()); err != nil {
		return errors.Wrap(err, "invalid compliance address")
	}
	if _, err := ToWei(r.GetInitialSupply(), 18); err != nil {
		return errors.Wrapf(err, "invalid inital supply(=%v)", r.GetInitialSupply())
	}
	return nil
}

func (r *IssueRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if err := validateAddress(r.GetRecipient()); err != nil {
		return errors.Wrap(err, "invalid recipient address")
	}
	if _, err := ToWei(r.GetAmount(), 18); err != nil {
		return errors.Wrapf(err, "invalid amount(=%v)", r.GetAmount())
	}
	return nil
}

func (r *TransferRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if err := validateAddress(r.GetRecipient()); err != nil {
		return errors.Wrap(err, "invalid recipient address")
	}
	if _, err := ToWei(r.GetAmount(), 18); err != nil {
		return errors.Wrapf(err, "invalid amount(=%v)", r.GetAmount())
	}
	return nil
}

func (r *RedeemRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if err := validateAddress(r.GetAccount()); err != nil {
		return errors.Wrap(err, "invalid account address")
	}
	if _, err := ToWei(r.GetAmount(), 18); err != nil {
		return errors.Wrapf(err, "invalid amount(=%v)", r.GetAmount())
	}
	return nil
}

func (r *RegisterWalletRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if err := validateAddress(r.GetAccount()); err != nil {
		return errors.Wrap(err, "invalid account address")
	}
	return nil
}

func (r *GrantRoleRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if err := validateAddress(r.GetGrantee()); err != nil {
		return errors.Wrap(err, "invalid grantee address")
	}
	if _, err := hex.DecodeString(r.GetRole()); err != nil {
		return errors.Wrapf(err, "failed to decode role(=%s)", r.GetRole())
	}
	return nil
}

func (r *NameRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	return nil
}

func (r *SymbolRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	return nil
}

func (r *TotalSupplyRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	return nil
}

func (r *BalanceOfRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if err := validateAddress(r.GetAccount()); err != nil {
		return errors.Wrap(err, "invalid account address")
	}
	return nil
}

func (r *HasRoleRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if err := validateAddress(r.GetAccount()); err != nil {
		return errors.Wrap(err, "invalid account address")
	}
	if _, err := hex.DecodeString(r.GetRole()); err != nil {
		return errors.Wrapf(err, "failed to decode role(=%s)", r.GetRole())
	}
	return nil
}

func (r *CreateContractsRequest) Validate() error {
	if err := validateAddress(r.GetContractAddress()); err != nil {
		return errors.Wrap(err, "invalid contract address")
	}
	if _, err := ToWei(r.GetInitialSupply(), 18); err != nil {
		return errors.Wrapf(err, "invalid inital supply(=%v)", r.GetInitialSupply())
	}
	for i, grantee := range r.GetGrantees() {
		if err := validateAddress(grantee); err != nil {
			return errors.Wrapf(err, "invalid grantee address at index %d", i)
		}
	}
	return nil
}

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

func validateAddress(address string) error {
	if address == "0x0000000000000000000000000000000000000000" || address == "0000000000000000000000000000000000000000" {
		return errors.New("empty ethereum address")
	}
	if !common.IsHexAddress(address) {
		return errors.Errorf("invalid ethereum address")
	}
	return nil
}
