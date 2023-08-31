package cheqd_interchaintest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	//"github.com/strangelove-ventures/interchaintest/v6/conformance"
	//"github.com/strangelove-ventures/interchaintest/v6/ibc"
	//"github.com/strangelove-ventures/interchaintest/v6/relayer"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/strangelove-ventures/interchaintest/v6/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/cosmos/cosmos-sdk/x/upgrade/plan"
)

const (
	haltHeightDelta    = uint64(10) // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = uint64(10)
)

func TestCheqdUpgradeIBC(t *testing.T) {
	CosmosChainUpgradeIBCTest(t, "cheqd", "v1.4.4-heighliner", "ghcr.io/nymlab/cheqd-node", "v2.0.0-rc1-heighliner", "v2")
}

func CosmosChainUpgradeIBCTest(t *testing.T, chainName string, initialVersion string, initialContainerRepo string, upgradeVersion string, upgradeName string) {

	t.Parallel()

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:        chainName,
			ChainName:   chainName,
			Version:     initialVersion,
			ChainConfig: GetCheqdConfig(initialVersion),
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	client, network := interchaintest.DockerSetup(t)

	chain := chains[0].(*cosmos.CosmosChain)

	const (
		path        = "ibc-upgrade-test-path"
		relayerName = "relayer"
	)

	//// Get a relayer instance
	//rf := interchaintest.NewBuiltinRelayerFactory(
	//	ibc.CosmosRly,
	//	zaptest.NewLogger(t),
	//	relayer.StartupFlags("-b", "100"),
	//)

	//r := rf.Build(t, client, network)

	ic := interchaintest.NewInterchain().
		AddChain(chain)

	ctx := context.Background()

	rep := testreporter.NewNopReporter()

	require.NoError(t, ic.Build(ctx, rep.RelayerExecReporter(t), interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	}))

	t.Cleanup(func() {
		_ = ic.Close()
	})

	const userFunds = int64(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, chain)
	chainUser := users[0]

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta

	urlMap := make(plan.BinaryDownloadURLMap)
	urlMap["darwin/amd64"] = "https://v1.cosmos.network/sdk?checksum=sha256:b5a2c96250612366ea272ffac6d9744aaf4b45aacd96aa7cfcb931ee3b558259"

	upgradeInfo := plan.Info{
		Binaries: urlMap,
	}

	info, err := json.Marshal(upgradeInfo)
	if err != nil {
		fmt.Errorf("cannot marshal info")
	}

	proposal := cosmos.SoftwareUpgradeProposal{
		Deposit:     "500000000" + chain.Config().Denom, // greater than min deposit
		Title:       "Chain Upgrade 1",
		Name:        upgradeName,
		Description: "First chain software upgrade",
		Height:      haltHeight,
		Info:        string(info),
	}

	upgradeTx, err := chain.UpgradeProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err, "error submitting software upgrade proposal tx")

	err = chain.VoteOnProposalAllValidators(ctx, upgradeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+haltHeightDelta, upgradeTx.ProposalID, cosmos.ProposalStatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// this should timeout due to chain halt at upgrade height.
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chain)

	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")

	// make sure that chain is halted
	require.Equal(t, haltHeight, height, "height is not equal to halt height")

	// bring down nodes to prepare for upgrade
	err = chain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	chain.UpgradeVersion(ctx, client, upgradeVersion)

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	err = chain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

}
