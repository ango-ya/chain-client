package client

import (
	"testing"
	"time"

	"github.com/ango-ya/chain-client/data"
	"github.com/stretchr/testify/require"
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
	TestComplianceAddress    = "0xe868feADdAA8965b6e64BDD50a14cD41e3D5245D"
	TestSecurityTokenAddress = "0xA7E7717817776181f64b46f9e4EFC75e181f9Dce"
	TestFactoryAddress       = "0xEEC5a0C20EC8b587E11604597d7a51779e3a71F2"
)

func TestShutdown(t *testing.T) {
	c, err := NewBlockchainClient(TestEndpoint, WithTimeout(3))
	require.NoError(t, err)

	ln, err := c.Start()

	time.Sleep(1 * time.Second)

	c.Close(ln)
}

func TestDeploySecurityToken(t *testing.T) {
	var (
		c, _ = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		req  = data.DeploySTRequest{
			PrivateKey:        TestPrivKey,
			Name:              "Test Token Name",
			Symbol:            "TKN",
			InitialSupply:     "100",
			ComplianceAddress: TestComplianceAddress,
		}
	)
	ln, err := c.Start()
	defer c.Close(ln)

	res, err := c.DeploySecurityToken(req)
	require.NoError(t, err)

	var (
		supReq = data.TotalSupplyRequest{
			ContractAddress: res.GetContractAddress(),
		}
		expected, _ = data.ToWei(req.GetInitialSupply(), 18)
	)
	supRes, err := c.TotalSupplySecurityToken(supReq)
	require.NoError(t, err)
	require.Equal(t, expected.String(), supRes.GetAmount())
}

func TestIssueSecurityToken(t *testing.T) {
	var (
		c, _ = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		req  = data.RegisterWalletRequest{
			PrivateKey:      TestPrivKey2,
			ContractAddress: TestComplianceAddress,
			Account:         TestAccount3,
		}
	)
	ln, err := c.Start()
	defer c.Close(ln)

	_, err = c.RegisterWalletComplianceService(req)
	require.NoError(t, err)

	issReq := data.IssueRequest{
		PrivateKey:      TestPrivKey2,
		ContractAddress: TestSecurityTokenAddress,
		Recipient:       TestAccount3,
		Amount:          "100",
	}

	_, err = c.IssueSecurityToken(issReq)
	require.NoError(t, err)

	var (
		balReq = data.BalanceOfRequest{
			ContractAddress: TestSecurityTokenAddress,
			Account:         TestAccount3,
		}
		expected, _ = data.ToWei(issReq.GetAmount(), 18)
	)
	balRes, err := c.BalanceOfSecurityToken(balReq)
	require.NoError(t, err)
	require.Equal(t, expected.String(), balRes.GetAmount())
}

func TestComplianceService(t *testing.T) {
	var (
		c, _ = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		req  = data.DeployCSRequest{
			PrivateKey: TestPrivKey,
		}
	)
	ln, err := c.Start()
	defer c.Close(ln)

	res, err := c.DeployComplianceService(req)
	require.NoError(t, err)

	var (
		granteee = "0xBfCD2b748782b2e958C06Fecfc6D7093599ed8c8"
		gReq     = data.GrantRoleRequest{
			PrivateKey:      TestPrivKey,
			ContractAddress: res.GetContractAddress(),
			Role:            ST_CONTROL_ROLE,
			Grantee:         granteee,
		}
	)
	_, err = c.GrantRole(gReq)
	require.NoError(t, err)

	var (
		hasReq = data.HasRoleRequest{
			ContractAddress: res.GetContractAddress(),
			Role:            ST_CONTROL_ROLE,
			Account:         granteee,
		}
	)
	hasRes, err := c.HasRole(hasReq)
	require.NoError(t, err)
	require.True(t, hasRes.GetHas())
}

func TestFactory(t *testing.T) {
	var (
		c, _     = NewBlockchainClient(TestEndpoint, WithTimeout(3))
		grantees = []string{
			"0xBfCD2b748782b2e958C06Fecfc6D7093599ed8c8",
			"0xC9911Ccf8FacBA9D7D8f1C59FE477233b6Bb9fE4",
		}
		req = data.CreateContractsRequest{
			PrivateKey:      TestPrivKey,
			ContractAddress: TestFactoryAddress,
			Name:            "Test Token Name",
			Symbol:          "TKN",
			InitialSupply:   "100",
			Grantees:        grantees,
		}
	)
	ln, err := c.Start()
	defer c.Close(ln)

	res, err := c.CreateContracts(req)
	require.NoError(t, err)

	for _, granteee := range grantees {
		var (
			hasReq = data.HasRoleRequest{
				ContractAddress: res.GetComplianceAddress(),
				Role:            ST_CONTROL_ROLE,
				Account:         granteee,
			}
		)
		hasRes, err := c.HasRole(hasReq)
		require.NoError(t, err)
		require.True(t, hasRes.GetHas())
	}
}
