package cheqd_interchaintest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v6"
	//"github.com/strangelove-ventures/interchaintest/v6/conformance"
	//"github.com/strangelove-ventures/interchaintest/v6/ibc"
	//"github.com/strangelove-ventures/interchaintest/v6/relayer"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/stretchr/testify/require"
	//"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/cosmos/cosmos-sdk/x/upgrade/plan"
	"github.com/strangelove-ventures/interchaintest/v6/testutil"
)

func TestCheqdUpgradeIBC(t *testing.T) {
	CosmosChainUpgradeIBCTest(t, "cheqd", "sha-5c98ec329797eb7fae0bc40e4b3090b3114e6c24", "ghcr.io/nymlab/cheqd-node", "v2.0.0-rc1", "v2")
}

func CosmosChainUpgradeIBCTest(t *testing.T, chainName string, initialVersion string, initialContainerRepo string, upgradeVersion string, upgradeName string) {

	//if testing.Short() {
	//	t.Skip("skipping in short mode")
	//}

	t.Parallel()
	ctx, cancelFn := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancelFn()
	})

	// create a single chain instance with x validators
	ic, chain, client, _ := CreateCheqdChain(t, ctx, 2, 1, "sha-5c98ec329797eb7fae0bc40e4b3090b3114e6c24")
	require.NotNil(t, ic)
	require.NotNil(t, chain)

	const userFunds = int64(10_000_000_000_000)
	chainUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, chain)
	chainUser := chainUsers[0]

	preUpgradeResource := CreateAndUploadDid(t, ctx, "did_payload.json", "resource_payload.json", "revocationList", chain, chainUser, "5rjaLzcffhGUH4nt4fyfAg", "9fbb1b86-91f8-4942-97b9-725b7714131c")

	//fmt.Println(preUpgradeResource)
	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta

	urlMap := make(plan.BinaryDownloadURLMap)
	// use a small file
	urlMap["darwin/amd64"] = "https://github.com/CosmWasm/cw-plus/releases/download/v1.1.0/cw4_group.wasm?checksum=sha256:506f9d6ebbe7350cc92620675e8e85988e052d4cede4d1e4d95f7201b8a0e223"

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

	t.Cleanup(func() {
		_ = ic.Close()
	})
}
