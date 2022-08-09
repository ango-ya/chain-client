package client

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ango-ya/chain-client/contract"
	"github.com/ango-ya/chain-client/data"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	svrdata "github.com/tak1827/fast-domain-socket-server/data"
)

func (c *BlockchainClient) handler(msg *svrdata.Message) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	switch msg.GetType() {
	case "SEND_ETH":
		return c.handleSendETH(ctx, msg.GetPayload())
	case "BALANCE_OF_ETH":
		return c.handleBalanceOfETH(ctx, msg.GetPayload())
	case "DEPLOY_ST":
		return c.handleDeploySecurityToken(ctx, msg.GetPayload())
	case "DEPLOY_CS":
		return c.handleDeployComplianceService(ctx, msg.GetPayload())
	case "ISSUE":
		return c.handleIssue(ctx, msg.GetPayload())
	case "TRANSFER":
		return c.handleTransfer(ctx, msg.GetPayload())
	case "REGISTER_WALLET":
		return c.handleRegisterWallet(ctx, msg.GetPayload())
	case "GRANT_ROLE":
		return c.handleGrantRole(ctx, msg.GetPayload())
	case "TOTAL_SUPPLY":
		return c.handleTotalSupply(ctx, msg.GetPayload())
	case "BALANCE_OF":
		return c.handleBalanceOf(ctx, msg.GetPayload())
	case "HAS_ROLE":
		return c.handleHasRole(ctx, msg.GetPayload())
	case "DEPLOY_FC":
		return c.handleDeployFactory(ctx, msg.GetPayload())
	case "CREATE_CONTRACTS":
		return c.handleCreateContracts(ctx, msg.GetPayload())
	default:
	}

	return nil, fmt.Errorf("unsupported message type(=%s)", msg.GetType())
}

func (c *BlockchainClient) errHandler(err error) {
	c.logger.Error().Stack().Err(err).Msg("at BlockchainClient.errHandler")
}

func (c *BlockchainClient) handleSendETH(ctx context.Context, payload string) ([]byte, error) {
	var req data.SendETHRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall SendETHRequest")
	}

	amount, err := data.ToWei(req.GetAmount(), 18)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid amount(=%v)", req.GetAmount())
	}

	var (
		recipient = common.HexToAddress(req.GetRecipient())
	)

	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &recipient, amount, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send transaction")
	}

	c.logger.Info().Msgf("eth sent, amount=%s, recipient=%s", req.GetAmount(), req.GetRecipient())

	resp := data.SendETHResponse{
		Hash: hash,
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleBalanceOfETH(ctx context.Context, payload string) ([]byte, error) {
	var req data.BalanceOfETHRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall BalanceOfETHRequest")
	}

	var (
		account = common.HexToAddress(req.GetAccount())
	)
	amount, err := c.ethclient.BalanceOf(ctx, account)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the balance of %s", req.GetAccount())
	}

	resp := data.BalanceOfETHResponse{
		Amount: amount.String(),
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleDeploySecurityToken(ctx context.Context, payload string) ([]byte, error) {
	var req data.DeploySTRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall DeploySTRequest")
	}

	initalSupply, err := data.ToWei(req.GetInitialSupply(), 18)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid inital supply(=%v)", req.GetInitialSupply())
	}

	var (
		complianceAddress = common.HexToAddress(req.GetComplianceAddress())
		input, _          = c.stABI.Pack("", []interface{}{req.GetName(), req.GetSymbol(), initalSupply, complianceAddress}...)
		bytecode          = common.FromHex(contract.SecurityTokenBin)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), nil, nil, append(bytecode, input...))
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send deploy transaction")
	}

	receipt, err := c.ethclient.Receipt(ctx, hash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the receipt of deployed transaction(=%s)", hash)
	}

	c.logger.Info().Msgf("contract deployed, name=%s, symbol=%s, supply=%s, compliance=%s, contract=%s", req.GetName(), req.GetSymbol(), req.GetInitialSupply(), req.GetComplianceAddress(), receipt.ContractAddress.String())

	resp := data.DeploySTResponse{
		Hash:            hash,
		ContractAddress: receipt.ContractAddress.String(),
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleDeployComplianceService(ctx context.Context, payload string) ([]byte, error) {
	var req data.DeployCSRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall DeployCSRequest")
	}

	var (
		bytecode = common.FromHex(contract.ComplianceServiceBin)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), nil, nil, bytecode)
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send deploy transaction")
	}

	receipt, err := c.ethclient.Receipt(ctx, hash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the receipt of deployed transaction(=%s)", hash)
	}

	c.logger.Info().Msgf("contract deployed, contract=%s", receipt.ContractAddress.String())

	resp := data.DeployCSResponse{
		Hash:            hash,
		ContractAddress: receipt.ContractAddress.String(),
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleIssue(ctx context.Context, payload string) ([]byte, error) {
	var req data.IssueRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall IssueRequest")
	}

	amount, err := data.ToWei(req.GetAmount(), 18)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid amount(=%v)", req.GetAmount())
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		recipient       = common.HexToAddress(req.GetRecipient())
		input, _        = c.stABI.Pack("issue", []interface{}{recipient, amount}...)
	)

	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &contractAddress, nil, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send issue transaction")
	}

	c.logger.Info().Msgf("token issued, amount=%s, recipient=%s, contract=%s", req.GetAmount(), req.GetRecipient(), req.GetContractAddress())

	resp := data.IssueResponse{
		Hash: hash,
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleTransfer(ctx context.Context, payload string) ([]byte, error) {
	var req data.TransferRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall TransferRequest")
	}

	amount, err := data.ToWei(req.GetAmount(), 18)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid amount(=%v)", req.GetAmount())
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		recipient       = common.HexToAddress(req.GetRecipient())
		input, _        = c.stABI.Pack("transfer", []interface{}{recipient, amount}...)
	)

	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &contractAddress, nil, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send transfer transaction")
	}

	c.logger.Info().Msgf("token transferd, amount=%s, recipient=%s, contract=%s", req.GetAmount(), req.GetRecipient(), req.GetContractAddress())

	resp := data.TransferResponse{
		Hash: hash,
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleRegisterWallet(ctx context.Context, payload string) ([]byte, error) {
	var req data.RegisterWalletRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall RegisterWalletRequest")
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		account         = common.HexToAddress(req.GetAccount())
		input, _        = c.csABI.Pack("registerWallet", []interface{}{account}...)
	)

	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &contractAddress, nil, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send register wallet transaction")
	}

	c.logger.Info().Msgf("wallet registerd, account=%s contract=%s", req.GetAccount(), req.GetContractAddress())

	resp := data.RegisterWalletResponse{
		Hash: hash,
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleGrantRole(ctx context.Context, payload string) ([]byte, error) {
	var (
		req  data.GrantRoleRequest
		role [32]byte
	)
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall GrantRoleRequest")
	}

	hexRole, err := hex.DecodeString(req.GetRole())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode role(=%s)", req.GetRole())
	}

	copy(role[:], hexRole)

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		grantee         = common.HexToAddress(req.GetGrantee())
		input, _        = c.csABI.Pack("setupRole", []interface{}{role, grantee}...)
	)

	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &contractAddress, nil, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send deploy transaction")
	}

	c.logger.Info().Msgf("wallet registerd, role=%s, grantee=%s contract=%s", req.GetRole(), req.GetGrantee(), req.GetContractAddress())

	resp := data.GrantRoleResponse{
		Hash: hash,
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleTotalSupply(ctx context.Context, payload string) ([]byte, error) {
	var req data.TotalSupplyRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall TotalSupplyRequest")
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		input, _        = c.stABI.Pack("totalSupply", []interface{}{}...)
	)
	output, err := c.ethclient.QueryContract(ctx, contractAddress, input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query contract(=%s), input(=%v)", contractAddress.String(), input)
	}

	var (
		results, _ = c.stABI.Unpack("totalSupply", output)
		amount     = *abi.ConvertType(results[0], new(*big.Int)).(**big.Int)
	)
	resp := data.TotalSupplyResponse{
		Amount: amount.String(),
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleBalanceOf(ctx context.Context, payload string) ([]byte, error) {
	var req data.BalanceOfRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall BalanceOfRequest")
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		acount          = common.HexToAddress(req.GetAccount())
		input, _        = c.stABI.Pack("balanceOf", []interface{}{acount}...)
	)
	output, err := c.ethclient.QueryContract(ctx, contractAddress, input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query contract(=%s), input(=%v)", contractAddress.String(), input)
	}

	var (
		results, _ = c.stABI.Unpack("balanceOf", output)
		amount     = *abi.ConvertType(results[0], new(*big.Int)).(**big.Int)
	)
	resp := data.BalanceOfResponse{
		Amount: amount.String(),
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleHasRole(ctx context.Context, payload string) ([]byte, error) {
	var (
		req  data.HasRoleRequest
		role [32]byte
	)
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall HasRoleRequest")
	}

	hexRole, err := hex.DecodeString(req.GetRole())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode role(=%s)", req.GetRole())
	}

	copy(role[:], hexRole)

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		acount          = common.HexToAddress(req.GetAccount())
		input, _        = c.csABI.Pack("hasRole", []interface{}{role, acount}...)
	)
	output, err := c.ethclient.QueryContract(ctx, contractAddress, input)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query contract(=%s), input(=%v)", contractAddress.String(), input)
	}

	var (
		results, _ = c.csABI.Unpack("hasRole", output)
		has        = *abi.ConvertType(results[0], new(bool)).(*bool)
	)
	resp := data.HasRoleResponse{
		Has: has,
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleDeployFactory(ctx context.Context, payload string) ([]byte, error) {
	var req data.DeployFCRequest
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall DeployFCRequest")
	}

	var (
		bytecode = common.FromHex(contract.FactoryV0Bin)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), nil, nil, bytecode)
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send deploy transaction")
	}

	receipt, err := c.ethclient.Receipt(ctx, hash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the receipt of deployed transaction(=%s)", hash)
	}

	c.logger.Info().Msgf("contract deployed, contract=%s", receipt.ContractAddress.String())

	resp := data.DeployFCResponse{
		Hash:            hash,
		ContractAddress: receipt.ContractAddress.String(),
	}

	return resp.Marshal()
}

func (c *BlockchainClient) handleCreateContracts(ctx context.Context, payload string) ([]byte, error) {
	var (
		req      data.CreateContractsRequest
		grantees = []common.Address{}
	)
	if err := req.Unmarshal([]byte(payload)); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshall CreateContractsRequest")
	}

	initalSupply, err := data.ToWei(req.GetInitialSupply(), 18)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid inital supply(=%v)", req.GetInitialSupply())
	}

	for _, grantee := range req.GetGrantees() {
		grantees = append(grantees, common.HexToAddress(grantee))
	}

	var (
		contractAddress = common.HexToAddress(req.GetContractAddress())
		input, _        = c.fcABI.Pack("create", []interface{}{req.GetName(), req.GetSymbol(), initalSupply, grantees}...)
	)
	hash, err := c.ethclient.SyncSend(ctx, req.GetPrivateKey(), &contractAddress, nil, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed sync send deploy transaction")
	}

	receipt, err := c.ethclient.Receipt(ctx, hash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the receipt of deployed transaction(=%s)", hash)
	}

	clog := contract.FactoryV0Created{}
	if err := c.fcABI.UnpackIntoInterface(&clog, "Created", receipt.Logs[len(receipt.Logs)-1].Data); err != nil {
		return nil, errors.Wrapf(err, "failed unpack log transaction(=%v)", receipt.Logs)
	}

	c.logger.Info().Msgf("contract deployed, name=%s, symbol=%s, supply=%s, granteees=%v, compliance=%s, token=%s", req.GetName(), req.GetSymbol(), req.GetInitialSupply(), req.GetGrantees(), clog.Compliance.String(), clog.Token.String())

	resp := data.CreateContractsResponse{
		Hash:              hash,
		ComplianceAddress: clog.Compliance.String(),
		TokenAddress:      clog.Token.String(),
	}

	return resp.Marshal()
}
