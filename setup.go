package cheqd_interchaintest

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/icza/dyno"
	"testing"

	"github.com/docker/docker/client"
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

const (
	votingPeriod       = "10s"
	maxDepositPeriod   = "10s"
	haltHeightDelta    = uint64(10) // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = uint64(10)
)

func ModifyGenesisShortProposals(votingPeriod string, maxDepositPeriod string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "voting_params", "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "deposit_params", "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set max deposit period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "deposit_params", "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set min deposit denom in genesis json: %w", err)
		}
		if err := dyno.Set(g, "100", "app_state", "gov", "deposit_params", "min_deposit", 0, "amount"); err != nil {
			return nil, fmt.Errorf("failed to set min deposit amount in genesis json: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
func GetCheqdConfig(version string) ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "cheqd",
		ChainID: "cheqd-mainnet-1",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/nymlab/cheqd-node", // FOR LOCAL IMAGE USE: Docker Image Name
				Version:    version,                     // FOR LOCAL IMAGE USE: Docker Image Tag
				UidGid:     "1000:1000",
				//Repository: "ghcr.io/nymlab/cheqd-node", // FOR LOCAL IMAGE USE: Docker Image Name
				//Version:    version,                     // FOR LOCAL IMAGE USE: Docker Image Tag
				//UidGid:     "1025:1025",
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
		ModifyGenesis:       ModifyGenesisShortProposals(votingPeriod, maxDepositPeriod),
	}
}

func GetJunoConfig() ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "juno",
		ChainID: "juno-mainnet-1",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/strangelove-ventures/heighliner/juno",
				Version:    "v14.1.0",
				UidGid:     "1025:1025",
			},
		},
		Bin:                 "junod",
		Bech32Prefix:        "juno",
		Denom:               "ujunox",
		CoinType:            "118",
		GasPrices:           "50ujunox",
		GasAdjustment:       1.3,
		TrustingPeriod:      "508h",
		NoHostMount:         false,
		ConfigFileOverrides: nil,
		EncodingConfig:      nil,
		ModifyGenesis:       ModifyGenesisShortProposals(votingPeriod, maxDepositPeriod),
	}
}

func CreateCheqdChain(
	t *testing.T,
	ctx context.Context,
	numVals, numFull int,
	version string,
) (*interchaintest.Interchain, *cosmos.CosmosChain, *client.Client, string) {

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "cheqd",
			ChainName:     "cheqd",
			Version:       version,
			ChainConfig:   GetCheqdConfig(version),
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
