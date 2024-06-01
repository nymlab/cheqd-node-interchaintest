package cheqd_interchaintest

import (
	"context"
	_ "embed"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	ibclocalhost "github.com/cosmos/ibc-go/v7/modules/light-clients/09-localhost"
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

	clocktypes "github.com/CosmosContracts/juno/v22/x/clock/types"
	feepaytypes "github.com/CosmosContracts/juno/v22/x/feepay/types"
	feesharetypes "github.com/CosmosContracts/juno/v22/x/feeshare/types"
	globalfeetypes "github.com/CosmosContracts/juno/v22/x/globalfee/types"
	tokenfactorytypes "github.com/CosmosContracts/juno/v22/x/tokenfactory/types"
)

func cheqdEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	didtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	resourcetypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

//go:embed artifacts/jwk.json
var jwk []byte

const (
	userFunds          = int64(10_000_000_000_000)
	cheqdDenom         = "ncheq"
	junoDenom          = "ujuno"
	haltHeightDelta    = uint64(20) // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = uint64(10)
	ssiPath            = "ssi-cheqd-juno-path"
	path               = "cheqd-juno-path"
	relayerName        = "relayer"
	collectionId       = "5rjaLzcffhGUH4nt4fyfAg"
	resourceId         = "9fbb1b86-91f8-4942-97b9-725b7714131c"
	contractPath       = "contracts_wasm/avida_sdjwt_verifier.wasm"
)

func GetJunoEncoding() *testutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	// register custom types
	ibclocalhost.RegisterInterfaces(cfg.InterfaceRegistry)
	wasmtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	feesharetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	tokenfactorytypes.RegisterInterfaces(cfg.InterfaceRegistry)
	feepaytypes.RegisterInterfaces(cfg.InterfaceRegistry)
	globalfeetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	clocktypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

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

func GetJunoConfig() ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "juno",
		ChainID: "juno-mainnet-1",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/cosmoscontracts/juno",
				Version:    "v22.0.0",
				UidGid:     "1025:1025",
			},
		},
		Bin:                 "junod",
		Bech32Prefix:        "juno",
		Denom:               junoDenom,
		CoinType:            "118",
		GasPrices:           "50 ujuno",
		GasAdjustment:       1.3,
		TrustingPeriod:      "508h",
		NoHostMount:         false,
		ConfigFileOverrides: nil,
		EncodingConfig:      GetJunoEncoding(),
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
			NoHostMount:   &[]bool{false}[0], // specify no mount
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

	return ic, chains[0].(*cosmos.CosmosChain), client, network
}

func CreateJunoChain(
	t *testing.T,
	ctx context.Context,
	numVals, numFull int,
) (*interchaintest.Interchain, *cosmos.CosmosChain, *client.Client, string) {

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "juno",
			ChainName:     "juno",
			Version:       "v22.0.0",
			ChainConfig:   GetJunoConfig(),
			NoHostMount:   &[]bool{false}[0], // specify no mount
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

	return ic, chains[0].(*cosmos.CosmosChain), client, network
}
