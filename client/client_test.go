package client

import (
	"context"
	"testing"
	"time"

	"github.com/ango-ya/chain-client/data"
	"github.com/stretchr/testify/require"
	eclient "github.com/tak1827/eth-extended-client/client"
	// "go.uber.org/goleak"
)

const (
	TestEndpoint             = "http://localhost:8545"
	TestPrivKey              = "d1c71e71b06e248c8dbe94d49ef6d6b0d64f5d71b1e33a0f39e14dadb070304a"
	TestAccount              = "0xE3b0DE0E4CA5D3CB29A9341534226C4D31C9838f"
	TestPrivKey2             = "8179ce3d00ac1d1d1d38e4f038de00ccd0e0375517164ac5448e3acc847acb34"
	TestAccount2             = "0x26fa9f1a6568b42e29b1787c403B3628dFC0C6FE"
	TestPrivKey3             = "df38daebd09f56398cc8fd699b72f5ea6e416878312e1692476950f427928e7d"
	TestAccount3             = "0x31a6EE302c1E7602685c86EF7a3069210Bc26670"
	TestPrivKey4             = "97d12403ffc2faa3660730ae58bca14a894ebd78b4d8207d22083554ae96be5c"
	TestAccount4             = "0xa52ce7A3B18095800ed1f550065DF9Cd5ca5ce9f"
	TestComplianceAddress    = "0xe868feADdAA8965b6e64BDD50a14cD41e3D5245D"
	TestSecurityTokenAddress = "0xA7E7717817776181f64b46f9e4EFC75e181f9Dce"
)

func TestShutdown(t *testing.T) {
	c, err := NewBlockchainClient(TestEndpoint, WithTimeout(3))
	require.NoError(t, err)

	c.Start()

	time.Sleep(1 * time.Second)

	c.Close()
}

func TestETH(t *testing.T) {
	var (
		ctx          = context.Background()
		c, _         = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		recipient, _ = eclient.GenerateAddr()
		req          = data.SendETHRequest{
			PrivateKey: TestPrivKey3,
			Recipient:  recipient.String(),
			Amount:     "1",
		}
		expected = eclient.ToWei(1.0, 18).String()
	)
	c.Start()
	defer c.Close()

	_, err := c.SendETH(ctx, req)
	require.NoError(t, err)

	bReq := data.BalanceOfETHRequest{
		Account: recipient.String(),
	}
	bRes, err := c.BalanceOfETH(ctx, bReq)
	require.NoError(t, err)
	require.Equal(t, expected, bRes.GetAmount())
}

func TestDeploySecurityToken(t *testing.T) {
	var (
		ctx  = context.Background()
		c, _ = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		req  = data.DeploySTRequest{
			PrivateKey:        TestPrivKey,
			Name:              "Test Token Name",
			Symbol:            "TKN",
			InitialSupply:     "100",
			ComplianceAddress: TestComplianceAddress,
		}
	)
	c.Start()
	defer c.Close()

	res, err := c.DeploySecurityToken(ctx, req)
	require.NoError(t, err)

	var (
		supReq = data.TotalSupplyRequest{
			ContractAddress: res.GetContractAddress(),
		}
		expected, _ = data.ToWei(req.GetInitialSupply(), 18)
	)
	nameRes, err := c.NameSecurityToken(ctx, data.NameRequest(supReq))
	require.NoError(t, err)
	symRes, err := c.SymbolSecurityToken(ctx, data.SymbolRequest(supReq))
	require.NoError(t, err)
	supRes, err := c.TotalSupplySecurityToken(ctx, supReq)
	require.NoError(t, err)
	require.Equal(t, req.GetName(), nameRes.GetName())
	require.Equal(t, req.GetSymbol(), symRes.GetSymbol())
	require.Equal(t, expected.String(), supRes.GetAmount())
}

func TestIssueTransferSecurityToken(t *testing.T) {
	var (
		ctx  = context.Background()
		c, _ = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		// req  = data.RegisterWalletRequest{
		// 	PrivateKey:      TestPrivKey2,
		// 	ContractAddress: TestComplianceAddress,
		// 	Account:         TestAccount3,
		// }
	)
	c.Start()
	defer c.Close()

	// トークンの発行
	// _, err = c.RegisterWalletComplianceService(ctx, req)
	// require.NoError(t, err)

	_, err := c.IssueSecurityToken(ctx, data.IssueRequest{
		PrivateKey:      TestPrivKey2,
		ContractAddress: TestSecurityTokenAddress,
		Recipient:       TestAccount3,
		Amount:          "100",
	})
	require.NoError(t, err)

	var (
		balReq = data.BalanceOfRequest{
			ContractAddress: TestSecurityTokenAddress,
			Account:         TestAccount3,
		}
		expected, _ = data.ToWei("100", 18)
	)
	balRes, err := c.BalanceOfSecurityToken(ctx, balReq)
	require.NoError(t, err)
	require.Equal(t, expected.String(), balRes.GetAmount())

	// トークンの移転
	// req.Account = TestAccount4
	// _, err = c.RegisterWalletComplianceService(ctx, req)
	// require.NoError(t, err)

	_, err = c.TransferSecurityToken(ctx, data.TransferRequest{
		PrivateKey:      TestPrivKey3,
		ContractAddress: TestSecurityTokenAddress,
		Recipient:       TestAccount4,
		Amount:          "50",
	})
	require.NoError(t, err)

	balReq.Account = TestAccount4
	expected, _ = data.ToWei("50", 18)
	balRes, err = c.BalanceOfSecurityToken(ctx, balReq)
	require.NoError(t, err)
	require.Equal(t, expected.String(), balRes.GetAmount())
}

func TestComplianceService(t *testing.T) {
	var (
		ctx  = context.Background()
		c, _ = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		req  = data.DeployCSRequest{
			PrivateKey: TestPrivKey,
		}
	)
	c.Start()
	defer c.Close()

	res, err := c.DeployComplianceService(ctx, req)
	require.NoError(t, err)

	var (
		granteee = "0xBfCD2b748782b2e958C06Fecfc6D7093599ed8c8"
		gReq     = data.GrantRoleRequest{
			PrivateKey:      TestPrivKey,
			ContractAddress: res.GetContractAddress(),
			Role:            data.ST_CONTROL_ROLE,
			Grantee:         granteee,
		}
	)
	_, err = c.GrantRole(ctx, gReq)
	require.NoError(t, err)

	var (
		hasReq = data.HasRoleRequest{
			ContractAddress: res.GetContractAddress(),
			Role:            data.ST_CONTROL_ROLE,
			Account:         granteee,
		}
	)
	hasRes, err := c.HasRole(ctx, hasReq)
	require.NoError(t, err)
	require.True(t, hasRes.GetHas())
}

func TestFactory(t *testing.T) {
	var (
		ctx      = context.Background()
		c, _     = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		grantees = []string{
			"0xBfCD2b748782b2e958C06Fecfc6D7093599ed8c8",
			"0xC9911Ccf8FacBA9D7D8f1C59FE477233b6Bb9fE4",
		}
		req = data.DeployFCRequest{
			PrivateKey: TestPrivKey,
		}
	)
	c.Start()
	defer c.Close()

	res, err := c.DeployFactory(ctx, req)
	require.NoError(t, err)

	cReq := data.CreateContractsRequest{
		PrivateKey:      TestPrivKey,
		ContractAddress: res.GetContractAddress(),
		Name:            "Test Token Name",
		Symbol:          "TKN",
		InitialSupply:   "100",
		Grantees:        grantees,
	}

	cRes, err := c.CreateContracts(ctx, cReq)
	require.NoError(t, err)

	for _, granteee := range grantees {
		var (
			hasReq = data.HasRoleRequest{
				ContractAddress: cRes.GetComplianceAddress(),
				Role:            data.ST_CONTROL_ROLE,
				Account:         granteee,
			}
		)
		hasRes, err := c.HasRole(ctx, hasReq)
		require.NoError(t, err)
		require.True(t, hasRes.GetHas())
	}
}

func TestAsyncSend(t *testing.T) {
	var (
		ctx  = context.Background()
		c, _ = NewBlockchainClient(TestEndpoint, WithTimeout(3))
	)
	c.Start()
	defer c.Close()

	// 正常に終了する想定
	req := data.RegisterWalletRequest{
		PrivateKey:      TestPrivKey2,
		ContractAddress: TestComplianceAddress,
		Account:         TestAccount3,
		IsAsync:         true,
	}
	_, err := c.RegisterWalletComplianceService(ctx, req)
	require.NoError(t, err)

	// 正常に終了するが、トランザクション自体は失敗する
	req = data.RegisterWalletRequest{
		PrivateKey:      TestPrivKey2,
		ContractAddress: TestComplianceAddress,
		Account:         TestAccount3,
		IsAsync:         true,
		GasLimit:        23000,
	}
	_, err = c.RegisterWalletComplianceService(ctx, req)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	// panic("confirm that error log output")
}
