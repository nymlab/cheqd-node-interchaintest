package cheqd_interchaintest

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	interchaintest "github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"

	didtypes "github.com/cheqd/cheqd-node/x/did/types"
	resourcetypes "github.com/cheqd/cheqd-node/x/resource/types"
)

func cheqdEncoding() *simappparams.EncodingConfig {
	cfg := cosmos.DefaultEncoding()

	didtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	resourcetypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

func CreateChain(
	t *testing.T,
	ctx context.Context,
	numVals, numFull int,
) (*interchaintest.Interchain, *cosmos.CosmosChain) {

	cheqdConfig := ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "cheqd",
		ChainID: "cheqd-mainnet-1",
		Images: []ibc.DockerImage{
			{
				Repository: "cheqd",   // FOR LOCAL IMAGE USE: Docker Image Name
				Version:    "develop", // FOR LOCAL IMAGE USE: Docker Image Tag
				UidGid:     "1025:1025",
			},
		},
		Bin:                 "cheqd-noded",
		Bech32Prefix:        "cheqd",
		Denom:               "ncheq",
		CoinType:            "118",
		GasPrices:           "50ncheq",
		GasAdjustment:       1.3,
		TrustingPeriod:      "508h",
		NoHostMount:         false,
		ConfigFileOverrides: nil,
		EncodingConfig:      cheqdEncoding(),
	}

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "cheqd",
			ChainName:     "cheqd",
			Version:       "develop",
			ChainConfig:   cheqdConfig,
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

	fmt.Printf("error %s", err)
	require.NoError(t, err)

	return ic, chains[0].(*cosmos.CosmosChain)
}
