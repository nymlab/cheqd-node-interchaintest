package cheqd_interchaintest

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	testutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"

	didtypes "github.com/cheqd/cheqd-node/x/did/types"
	resourcetypes "github.com/cheqd/cheqd-node/x/resource/types"
)

func cheqdEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	didtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	resourcetypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

const (
	votingPeriod       = "10s"
	maxDepositPeriod   = "10s"
	cheqdDenom         = "ncheq"
	neutronDenom       = "untrn"
	haltHeightDelta    = uint64(20) // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = uint64(10)
)

func GetCheqdConfig() ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "cheqd",
		ChainID: "cheqd-mainnet-1",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/nymlab/cheqd-node", // FOR LOCAL IMAGE USE: Docker Image Name
				//Repository: "ghcr.io/strangelove-ventures/heighliner/cheqd", // FOR LOCAL IMAGE USE: Docker Image Name
				Version: "v2.0.1-arm64", // FOR LOCAL IMAGE USE: Docker Image Tag
				UidGid:  "1000:1000",
			},
		},
		Bin:                 "cheqd-noded",
		Bech32Prefix:        "cheqd",
		Denom:               cheqdDenom,
		CoinType:            "118",
		GasPrices:           "50ncheq",
		GasAdjustment:       1.3,
		TrustingPeriod:      "508h",
		NoHostMount:         false,
		ConfigFileOverrides: nil,
		EncodingConfig:      cheqdEncoding(),
	}
}

func GetNeutronConfig() ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "neutron",
		ChainID: "neutron-mainnet-1",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/strangelove-ventures/heighliner/neutron",
				Version:    "v2.0.4",
				UidGid:     "1025:1025",
			},
		},
		Bin:                 "neutrond",
		Bech32Prefix:        "neutron",
		Denom:               neutronDenom,
		CoinType:            "118",
		GasPrices:           "50 untrn",
		GasAdjustment:       1.3,
		TrustingPeriod:      "508h",
		NoHostMount:         false,
		ConfigFileOverrides: nil,
		EncodingConfig:      nil,
	}
}

func CreateCheqdChain(
	t *testing.T,
	ctx context.Context,
	numVals, numFull int,
) (*interchaintest.Interchain, *cosmos.CosmosChain, *client.Client, string) {

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "cheqd",
			ChainName:     "cheqd",
			Version:       "v2.0.1-arm64",
			ChainConfig:   GetCheqdConfig(),
			NoHostMount:   &[]bool{false}[0],
			NumValidators: &numVals,
			NumFullNodes:  &numFull,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	ic := interchaintest.NewInterchain().AddChain(chains[0])

	client, network := interchaintest.DockerSetup(t)

	err = ic.Build(
		ctx,
		testreporter.NewNopReporter().RelayerExecReporter(t),
		interchaintest.InterchainBuildOptions{
			TestName:         t.Name(),
			Client:           client,
			NetworkID:        network,
			SkipPathCreation: true,
		},
	)

	fmt.Print("building finished")

	require.NoError(t, err)

	return ic, chains[0].(*cosmos.CosmosChain), client, network
}
