package client

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/ango-ya/chain-client/contract"
	"github.com/ango-ya/chain-client/data"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	eclient "github.com/tak1827/eth-extended-client/client"
	srvdata "github.com/tak1827/fast-domain-socket-server/data"
	"github.com/tak1827/fast-domain-socket-server/server"
	"github.com/tak1827/transaction-confirmer/confirm"
)

const (
	SockFilePath = "./domain.sock"
)

var (
	timeoutDuration time.Duration
)

type BlockchainClient struct {
	ethclient eclient.Client
	srv       server.Server

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

	c.srv = server.NewServer(SockFilePath, server.WithTimeout(c.timeout), server.WithHandler(c.handler), server.WithErrHandler(c.errHandler))

	timeoutDuration = time.Duration(time.Duration(c.timeout) * time.Second)

	return
}

func (c *BlockchainClient) Start() (net.Listener, error) {
	ln, err := c.srv.Listen()
	if err != nil {
		return nil, err
	}

	go func() {
		if err = c.srv.Serve(ln); err != nil {
			c.logger.Error().Stack().Err(err).Msg("at domain.Serve")
		}
	}()

	c.ethclient.Start()

	return ln, nil
}

func (c *BlockchainClient) Close(ln net.Listener) {
	if err := c.srv.Shutdown(ln); err != nil {
		c.logger.Error().Stack().Err(err).Msg("failed to shut down domain server")
	}

	c.ethclient.Stop()

	return
}

func (c *BlockchainClient) SendETH(req data.SendETHRequest) (resp data.SendETHResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("SEND_ETH", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) BalanceOfETH(req data.BalanceOfETHRequest) (resp data.BalanceOfETHResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("BALANCE_OF_ETH", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) DeploySecurityToken(req data.DeploySTRequest) (resp data.DeploySTResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("DEPLOY_ST", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) DeployComplianceService(req data.DeployCSRequest) (resp data.DeployCSResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("DEPLOY_CS", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) IssueSecurityToken(req data.IssueRequest) (resp data.IssueResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("ISSUE", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) RegisterWalletComplianceService(req data.RegisterWalletRequest) (resp data.RegisterWalletResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("REGISTER_WALLET", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) GrantRole(req data.GrantRoleRequest) (resp data.GrantRoleResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("GRANT_ROLE", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) TotalSupplySecurityToken(req data.TotalSupplyRequest) (resp data.TotalSupplyResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("TOTAL_SUPPLY", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) BalanceOfSecurityToken(req data.BalanceOfRequest) (resp data.BalanceOfResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("BALANCE_OF", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) HasRole(req data.HasRoleRequest) (resp data.HasRoleResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("HAS_ROLE", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) DeployFactory(req data.DeployFCRequest) (resp data.DeployFCResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("DEPLOY_FC", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func (c *BlockchainClient) CreateContracts(req data.CreateContractsRequest) (resp data.CreateContractsResponse, err error) {
	b, err := req.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal")
		return
	}

	payload, err := request("CREATE_CONTRACTS", string(b))
	if err != nil {
		err = errors.Wrapf(err, "failed to request(=%v)", req)
		return
	}

	if err = resp.Unmarshal([]byte(payload)); err != nil {
		err = errors.Wrapf(err, "failed to unmarshall response(=%s)", payload)
		return
	}

	return
}

func request(mType, payload string) (response string, err error) {
	var (
		dst = make([]byte, 1024)
		msg = srvdata.Message{
			Type:    mType,
			Payload: payload,
		}
	)
	conn, err := net.Dial("unix", SockFilePath)
	if err != nil {
		return
	}
	defer conn.Close()

	b, err := msg.Marshal()
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal srvdata.Message(=%v)", msg)
		return
	}

	b = append(b, server.EOFByte)
	if _, err = conn.Write(b); err != nil {
		err = errors.Wrapf(err, "failed to write srvdata.Message(=%v)", msg)
		return
	}

	if dst, err = server.ReadConn(conn, dst); err != nil {
		err = errors.Wrap(err, "at server.ReadConn")
		return
	}

	response = string(dst)

	return
}
