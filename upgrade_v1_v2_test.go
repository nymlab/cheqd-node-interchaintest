package cheqd_interchaintest

import (
	"context"
	"encoding/json"
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

func TestCheqdUpgradeIBC(t *testing.T) {
	//CosmosChainUpgradeIBCTest(t, "cheqd", "v1.4.4-heighliner", "ghcr.io/nymlab/cheqd-node", "v2.0.0-rc1-heighliner", "v2")
	CosmosChainUpgradeIBCTest(t, "cheqd", "sha-5c98ec329797eb7fae0bc40e4b3090b3114e6c24", "ghcr.io/nymlab/cheqd-node", "v2.0.0-rc1", "v2")
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

	preUpgradeResource := CreateAndUploadDid(t, ctx, "did_payload.json", "resource_payload.json", "revocationList", chain, chainUser, "5rjaLzcffhGUH4nt4fyfAg", "9fbb1b86-91f8-4942-97b9-725b7714131c")

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta

	urlMap := make(plan.BinaryDownloadURLMap)
	// use a small file
	urlMap["darwin/amd64"] = "https://github.com/cheqd/did-resolver/releases/download/v3.5.1/did-resolver-3.5.1-darwin-amd64.tar.gz?checksum=sha256:3408e2a65fd1ccb57eee19ff8f42d6c50cc4876fbe91537864be2a335a9fa9dd"

	upgradeInfo := plan.Info{
		Binaries: urlMap,
	}

	info, err := json.Marshal(upgradeInfo)
	require.NoError(t, err, "error marshalling")

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

	err = chain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	res1, _, err := chain.Nodes()[0].ExecQuery(ctx, "resource", "specific-resource", "5rjaLzcffhGUH4nt4fyfAg", "9fbb1b86-91f8-4942-97b9-725b7714131c")
	require.NoError(t, err, "query-resource err")
	require.NotNil(t, res1, "return resource err")
	require.Equal(t, preUpgradeResource, res1)
}
