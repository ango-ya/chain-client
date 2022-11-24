package client

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/ango-ya/chain-client/contract"
	"github.com/ango-ya/chain-client/data"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	eclient "github.com/tak1827/eth-extended-client/client"
	"github.com/tak1827/transaction-confirmer/confirm"
)

var (
	timeoutDuration time.Duration
)

type BlockchainClient struct {
	ethclient eclient.Client

	stABI abi.ABI
	csABI abi.ABI
	fcABI abi.ABI

	timeout int64
	logger  zerolog.Logger
}

func NewBlockchainClient(endpoint string, opts ...Option) (c BlockchainClient, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	c.timeout = DefaultTimeout
	c.logger = DefaultLogger

	if c.stABI, err = abi.JSON(strings.NewReader(contract.SecurityTokenABI)); err != nil {
		return
	}

	if c.csABI, err = abi.JSON(strings.NewReader(contract.ComplianceServiceABI)); err != nil {
		return
	}

	if c.fcABI, err = abi.JSON(strings.NewReader(contract.FactoryV0ABI)); err != nil {
		return
	}

	for i := range opts {
		opts[i].Apply(&c)
	}

	cfmOpts := []confirm.Opt{
		confirm.WithWorkers(1),
		confirm.WithWorkerInterval(32),
		confirm.WithConfirmationBlock(0),
		confirm.WithTimeout(c.timeout),
	}

	ethOpts := []eclient.Option{
		eclient.WithLoggerOpt(c.logger),
		eclient.WithTimeout(c.timeout),
		eclient.WithSyncSendTimeout(c.timeout),
		eclient.WithSyncSendConfirmInterval(128),
	}

	if c.ethclient, err = eclient.NewClient(ctx, endpoint, cfmOpts, ethOpts...); err != nil {
		err = errors.Wrap(err, "failed to create eth client")
		return
	}

	timeoutDuration = time.Duration(time.Duration(c.timeout) * time.Second)

	return
}

func (c *BlockchainClient) Start() {
	c.ethclient.Start()
}

func (c *BlockchainClient) Close() {
	c.ethclient.Stop()
}

func (c *BlockchainClient) SendETH(ctx context.Context, req data.SendETHRequest) (resp data.SendETHResponse, err error) {
	amount, err := data.ToWei(req.GetAmount(), 18)
	if err != nil {
		err = errors.Wrapf(err, "invalid amount(=%v)", req.GetAmount())
		return
	}

	var (
		recipient = common.HexToAddress(req.GetRecipient())
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &recipient, amount, nil, 0)
	if err != nil {
		err = errors.Wrap(err, "failed sync send transaction")
		return
	}

	c.logger.Info().Msgf("eth sent, amount=%s, recipient=%s", req.GetAmount(), req.GetRecipient())

	resp = data.SendETHResponse{
		Hash: hash,
	}
	return
}

func (c *BlockchainClient) BalanceOfETH(ctx context.Context, req data.BalanceOfETHRequest) (resp data.BalanceOfETHResponse, err error) {
	var (
		account = common.HexToAddress(req.GetAccount())
	)
	amount, err := c.ethclient.BalanceOf(ctx, account)
	if err != nil {
		err = errors.Wrapf(err, "failed to get the balance of %s", req.GetAccount())
		return
	}

	resp = data.BalanceOfETHResponse{
		Amount: amount.String(),
	}
	return
}

func (c *BlockchainClient) DeploySecurityToken(ctx context.Context, req data.DeploySTRequest) (resp data.DeploySTResponse, err error) {
	initalSupply, err := data.ToWei(req.GetInitialSupply(), 18)
	if err != nil {
		err = errors.Wrapf(err, "invalid inital supply(=%v)", req.GetInitialSupply())
		return
	}

	var (
		complianceAddress = common.HexToAddress(req.GetComplianceAddress())
		input, _          = c.stABI.Pack("", []interface{}{req.GetName(), req.GetSymbol(), initalSupply, complianceAddress}...)
		bytecode          = common.FromHex(contract.SecurityTokenBin)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), nil, nil, append(bytecode, input...), 0)
	if err != nil {
		err = errors.Wrap(err, "failed sync send deploy transaction")
		return
	}

	receipt, err := c.ethclient.Receipt(ctx, hash)
	if err != nil {
		err = errors.Wrapf(err, "failed to get the receipt of deployed transaction(=%s)", hash)
		return
	}

	c.logger.Info().Msgf("contract deployed, name=%s, symbol=%s, supply=%s, compliance=%s, contract=%s", req.GetName(), req.GetSymbol(), req.GetInitialSupply(), req.GetComplianceAddress(), receipt.ContractAddress.String())

	resp = data.DeploySTResponse{
		Hash:            hash,
		ContractAddress: receipt.ContractAddress.String(),
	}
	return
}

func (c *BlockchainClient) DeployComplianceService(ctx context.Context, req data.DeployCSRequest) (resp data.DeployCSResponse, err error) {
	var (
		bytecode = common.FromHex(contract.ComplianceServiceBin)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), nil, nil, bytecode, 0)
	if err != nil {
		err = errors.Wrap(err, "failed sync send deploy transaction")
		return
	}

	receipt, err := c.ethclient.Receipt(ctx, hash)
	if err != nil {
		err = errors.Wrapf(err, "failed to get the receipt of deployed transaction(=%s)", hash)
		return
	}

	c.logger.Info().Msgf("contract deployed, contract=%s", receipt.ContractAddress.String())

	resp = data.DeployCSResponse{
		Hash:            hash,
		ContractAddress: receipt.ContractAddress.String(),
	}
	return
}

func (c *BlockchainClient) IssueSecurityToken(ctx context.Context, req data.IssueRequest) (resp data.IssueResponse, err error) {
	amount, err := data.ToWei(req.GetAmount(), 18)
	if err != nil {
		err = errors.Wrapf(err, "invalid amount(=%v)", req.GetAmount())
		return
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		recipient       = common.HexToAddress(req.GetRecipient())
		input, _        = c.stABI.Pack("issue", []interface{}{recipient, amount}...)
	)
	hash, err := c.send(ctx, req.GetPrivateKey(), &contractAddress, nil, input, req.GetGasLimit(), req.GetIsAsync())
	if err != nil {
		err = errors.Wrapf(err, "faile to send token issue transaction. contract=%s", req.GetContractAddress())
		return
	}

	c.logger.Info().Msgf("token issued, amount=%s, recipient=%s, contract=%s", req.GetAmount(), req.GetRecipient(), req.GetContractAddress())

	resp = data.IssueResponse{
		Hash: hash,
	}
	return
}

func (c *BlockchainClient) TransferSecurityToken(ctx context.Context, req data.TransferRequest) (resp data.TransferResponse, err error) {
	amount, err := data.ToWei(req.GetAmount(), 18)
	if err != nil {
		err = errors.Wrapf(err, "invalid amount(=%v)", req.GetAmount())
		return
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		recipient       = common.HexToAddress(req.GetRecipient())
		input, _        = c.stABI.Pack("transfer", []interface{}{recipient, amount}...)
	)

	hash, err := c.send(ctx, req.GetPrivateKey(), &contractAddress, nil, input, req.GetGasLimit(), req.GetIsAsync())
	if err != nil {
		err = errors.Wrapf(err, "faile to send token transfer transaction. contract=%s", req.GetContractAddress())
		return
	}

	c.logger.Info().Msgf("token transferd, amount=%s, recipient=%s, contract=%s", req.GetAmount(), req.GetRecipient(), req.GetContractAddress())

	resp = data.TransferResponse{
		Hash: hash,
	}
	return
}

func (c *BlockchainClient) BurnSecurityToken(ctx context.Context, req data.RedeemRequest) (resp data.RedeemResponse, err error) {
	amount, err := data.ToWei(req.GetAmount(), 18)
	if err != nil {
		err = errors.Wrapf(err, "invalid amount(=%v)", req.GetAmount())
		return
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		account         = common.HexToAddress(req.GetAccount())
		input, _        = c.stABI.Pack("redeem", []interface{}{account, amount, req.GetReason()}...)
	)
	hash, err := c.send(ctx, req.GetPrivateKey(), &contractAddress, nil, input, 0, false)
	if err != nil {
		err = errors.Wrapf(err, "faile to send token burn transaction. contract=%s", req.GetContractAddress())
		return
	}

	c.logger.Info().Msgf("token burned, amount=%s, account=%s, contract=%s", req.GetAmount(), req.GetAccount(), req.GetContractAddress())

	resp = data.RedeemResponse{
		Hash: hash,
	}
	return
}

func (c *BlockchainClient) RegisterWalletComplianceService(ctx context.Context, req data.RegisterWalletRequest) (resp data.RegisterWalletResponse, err error) {
	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		account         = common.HexToAddress(req.GetAccount())
		input, _        = c.csABI.Pack("registerWallet", []interface{}{account}...)
	)
	hash, err := c.send(ctx, req.GetPrivateKey(), &contractAddress, nil, input, req.GetGasLimit(), req.GetIsAsync())
	if err != nil {
		err = errors.Wrapf(err, "faile to send register wallet transaction. contract=%s", req.GetContractAddress())
		return
	}

	c.logger.Info().Msgf("wallet registerd, account=%s contract=%s", req.GetAccount(), req.GetContractAddress())

	resp = data.RegisterWalletResponse{
		Hash: hash,
	}
	return
}

func (c *BlockchainClient) GrantRole(ctx context.Context, req data.GrantRoleRequest) (resp data.GrantRoleResponse, err error) {
	hexRole, err := hex.DecodeString(req.GetRole())
	if err != nil {
		err = errors.Wrapf(err, "failed to decode role(=%s)", req.GetRole())
		return
	}

	var role [32]byte
	copy(role[:], hexRole)

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		grantee         = common.HexToAddress(req.GetGrantee())
		input, _        = c.csABI.Pack("setupRole", []interface{}{role, grantee}...)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &contractAddress, nil, input, 0)
	if err != nil {
		err = errors.Wrapf(err, "failed sync send grant role transaction. contract=%s", req.GetContractAddress())
		return
	}

	c.logger.Info().Msgf("wallet registerd, role=%s, grantee=%s contract=%s", req.GetRole(), req.GetGrantee(), req.GetContractAddress())

	resp = data.GrantRoleResponse{
		Hash: hash,
	}
	return
}

func (c *BlockchainClient) TotalSupplySecurityToken(ctx context.Context, req data.TotalSupplyRequest) (resp data.TotalSupplyResponse, err error) {
	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		input, _        = c.stABI.Pack("totalSupply", []interface{}{}...)
	)
	output, err := c.ethclient.QueryContract(ctx, contractAddress, input)
	if err != nil {
		err = errors.Wrapf(err, "failed to query contract(=%s), input(=%v)", contractAddress.String(), input)
		return
	}

	var (
		results, _ = c.stABI.Unpack("totalSupply", output)
		amount     = *abi.ConvertType(results[0], new(*big.Int)).(**big.Int)
	)
	resp = data.TotalSupplyResponse{
		Amount: amount.String(),
	}
	return
}

func (c *BlockchainClient) BalanceOfSecurityToken(ctx context.Context, req data.BalanceOfRequest) (resp data.BalanceOfResponse, err error) {
	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		acount          = common.HexToAddress(req.GetAccount())
		input, _        = c.stABI.Pack("balanceOf", []interface{}{acount}...)
	)
	output, err := c.ethclient.QueryContract(ctx, contractAddress, input)
	if err != nil {
		err = errors.Wrapf(err, "failed to query contract(=%s), input(=%v)", contractAddress.String(), input)
		return
	}

	var (
		results, _ = c.stABI.Unpack("balanceOf", output)
		amount     = *abi.ConvertType(results[0], new(*big.Int)).(**big.Int)
	)
	resp = data.BalanceOfResponse{
		Amount: amount.String(),
	}
	return
}

func (c *BlockchainClient) HasRole(ctx context.Context, req data.HasRoleRequest) (resp data.HasRoleResponse, err error) {
	hexRole, err := hex.DecodeString(req.GetRole())
	if err != nil {
		err = errors.Wrapf(err, "failed to decode role(=%s)", req.GetRole())
		return
	}

	var role [32]byte
	copy(role[:], hexRole)

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		acount          = common.HexToAddress(req.GetAccount())
		input, _        = c.csABI.Pack("hasRole", []interface{}{role, acount}...)
	)
	output, err := c.ethclient.QueryContract(ctx, contractAddress, input)
	if err != nil {
		err = errors.Wrapf(err, "failed to query contract(=%s), input(=%v)", contractAddress.String(), input)
		return
	}

	var (
		results, _ = c.csABI.Unpack("hasRole", output)
		has        = *abi.ConvertType(results[0], new(bool)).(*bool)
	)
	resp = data.HasRoleResponse{
		Has: has,
	}
	return
}

func (c *BlockchainClient) DeployFactory(ctx context.Context, req data.DeployFCRequest) (resp data.DeployFCResponse, err error) {
	var (
		bytecode = common.FromHex(contract.FactoryV0Bin)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), nil, nil, bytecode, 0)
	if err != nil {
		err = errors.Wrap(err, "failed sync send deploy transaction")
		return
	}

	receipt, err := c.ethclient.Receipt(ctx, hash)
	if err != nil {
		err = errors.Wrapf(err, "failed to get the receipt of deployed transaction(=%s)", hash)
		return
	}

	c.logger.Info().Msgf("contract deployed, contract=%s", receipt.ContractAddress.String())

	resp = data.DeployFCResponse{
		Hash:            hash,
		ContractAddress: receipt.ContractAddress.String(),
	}
	return
}

func (c *BlockchainClient) CreateContracts(ctx context.Context, req data.CreateContractsRequest) (resp data.CreateContractsResponse, err error) {
	initalSupply, err := data.ToWei(req.GetInitialSupply(), 18)
	if err != nil {
		err = errors.Wrapf(err, "invalid inital supply(=%v)", req.GetInitialSupply())
		return
	}

	grantees := []common.Address{}
	for _, grantee := range req.GetGrantees() {
		grantees = append(grantees, common.HexToAddress(grantee))
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		input, _        = c.fcABI.Pack("create", []interface{}{req.GetName(), req.GetSymbol(), initalSupply, grantees}...)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &contractAddress, nil, input, 0)
	if err != nil {
		err = errors.Wrap(err, "failed sync send deploy transaction")
		return
	}

	receipt, err := c.ethclient.Receipt(ctx, hash)
	if err != nil {
		err = errors.Wrapf(err, "failed to get the receipt of deployed transaction(=%s)", hash)
		return
	}

	clog := contract.FactoryV0Created{}
	if err = c.fcABI.UnpackIntoInterface(&clog, "Created", receipt.Logs[len(receipt.Logs)-1].Data); err != nil {
		err = errors.Wrapf(err, "failed unpack log transaction(=%v)", receipt.Logs)
		return
	}

	c.logger.Info().Msgf("contract deployed, name=%s, symbol=%s, supply=%s, granteees=%v, compliance=%s, token=%s", req.GetName(), req.GetSymbol(), req.GetInitialSupply(), req.GetGrantees(), clog.Compliance.String(), clog.Token.String())

	resp = data.CreateContractsResponse{
		Hash:              hash,
		ComplianceAddress: clog.Compliance.String(),
		TokenAddress:      clog.Token.String(),
	}
	return
}

func (c *BlockchainClient) send(ctx context.Context, priv string, to *common.Address, amount *big.Int, input []byte, gasLimit uint64, isAsync bool) (hash string, err error) {
	if !isAsync {
		if hash, err = c.ethclient.SyncSend(ctx, priv, to, amount, input, gasLimit); err != nil {
			err = errors.Wrap(err, "failed sync sending")
		}
		return
	}

	if hash, err = c.ethclient.AsyncSend(ctx, priv, to, amount, input, gasLimit); err != nil {
		err = errors.Wrap(err, "failed async sending")
		return
	}

	if err = c.ethclient.EnqueueTxHash(ctx, hash); err != nil {
		err = errors.Wrapf(err, "failed to enqueu async transaction(=%s)", hash)
		return
	}
	return
}
