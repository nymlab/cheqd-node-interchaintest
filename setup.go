package cheqd_interchaintest

import (
	"context"
	//"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	testutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	feesharetypes "github.com/CosmosContracts/juno/v16/x/feeshare/types"
	tokenfactorytypes "github.com/CosmosContracts/juno/v16/x/tokenfactory/types"
	ibclocalhost "github.com/cosmos/ibc-go/v7/modules/light-clients/09-localhost"
)

var (
	VotingPeriod     = "15s"
	MaxDepositPeriod = "10s"
	Denom            = "ujuno"
	JunoImage        = ibc.DockerImage{
		Repository: "ghcr.io/strangelove-ventures/heighliner/juno",
		Version:    "v16.0.0",
		UidGid:     "1025:1025",
	}
	// SDK v47 Genesis
	defaultGenesisKV = []cosmos.GenesisKV{
		{
			Key:   "app_state.gov.params.voting_period",
			Value: VotingPeriod,
		},
		{
			Key:   "app_state.gov.params.max_deposit_period",
			Value: MaxDepositPeriod,
		},
		{
			Key:   "app_state.gov.params.min_deposit.0.denom",
			Value: Denom,
		},
	}
)

func junoEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// register custom types
	ibclocalhost.RegisterInterfaces(cfg.InterfaceRegistry)
	wasmtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	feesharetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	tokenfactorytypes.RegisterInterfaces(cfg.InterfaceRegistry)

	//github.com/cosmos/cosmos-sdk/types/module/testutil

	return &cfg
}

func CreateChain(
	t *testing.T,
	ctx context.Context,
	numVals, numFull int,
	genesisOverrides ...cosmos.GenesisKV,
) (*interchaintest.Interchain, *cosmos.CosmosChain) {

	//junoConfig := ibc.ChainConfig{
	//	Type:                   "cosmos",
	//	Name:                   "juno",
	//	ChainID:                "juno-2",
	//	Images:                 []ibc.DockerImage{JunoImage},
	//	Bin:                    "junod",
	//	Bech32Prefix:           "juno",
	//	Denom:                  Denom,
	//	CoinType:               "118",
	//	GasPrices:              fmt.Sprintf("0%s", Denom),
	//	GasAdjustment:          2.0,
	//	TrustingPeriod:         "112h",
	//	NoHostMount:            false,
	//	ConfigFileOverrides:    nil,
	//	EncodingConfig:         junoEncoding(),
	//	UsingNewGenesisCommand: true,
	//	ModifyGenesis:          cosmos.ModifyGenesis(defaultGenesisKV),
	//}

	cheqdConfig := ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "cheqd",
		ChainID: "cheqd-mainnet-1",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/nymlab/cheqd", // FOR LOCAL IMAGE USE: Docker Image Name
				Version:    "prop31",               // FOR LOCAL IMAGE USE: Docker Image Tag
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
	}

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		//{
		//	Name:          "juno",
		//	ChainName:     "juno",
		//	Version:       "v16.0.0",
		//	ChainConfig:   junoConfig,
		//	NumValidators: &numVals,
		//	NumFullNodes:  &numFull,
		//},
		{
			Name:          "cheqd",
			ChainName:     "cheqd",
			Version:       "prop31",
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
	require.NoError(t, err)

	return ic, chains[0].(*cosmos.CosmosChain)
}
