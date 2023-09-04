package cheqd_interchaintest

import (
	"context"
	//	"fmt"
	//"time"
	//	"encoding/json"
	//	"os"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	//"github.com/strangelove-ventures/interchaintest/v6/conformance"
	//"github.com/strangelove-ventures/interchaintest/v6/ibc"
	//"github.com/strangelove-ventures/interchaintest/v6/relayer"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	//"github.com/strangelove-ventures/interchaintest/v6/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	//"github.com/cosmos/cosmos-sdk/x/upgrade/plan"
)

func TestCheqdProp31IBC(t *testing.T) {
	UpgradeV2ResourceIBCTest(t, "cheqd", "sha-5c98ec329797eb7fae0bc40e4b3090b3114e6c24", "ghcr.io/nymlab/cheqd-node", "v2.0.0-rc1", "v2")
}

func UpgradeV2ResourceIBCTest(t *testing.T, chainName string, initialVersion string, initialContainerRepo string, upgradeVersion string, upgradeName string) {

	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:        chainName,
			ChainName:   chainName,
			Version:     initialVersion,
			ChainConfig: GetCheqdConfig(initialVersion),
		},
		{
			Name:    "juno",
			Version: "v15.0.0",
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	client, network := interchaintest.DockerSetup(t)
	cheqd := chains[0].(*cosmos.CosmosChain)
	//cheqd, juno := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

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
		AddChain(cheqd)
		//AddChain(juno).
		//AddRelayer(r, relayerName).
		//AddLink(interchaintest.InterchainLink{
		//	Chain1:  cheqd,
		//	Chain2:  juno,
		//	Relayer: r,
		//	Path:    path,
		//})

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
	cheqdUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, cheqd)
	cheqdUser := cheqdUsers[0]

	//junoUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, juno)
	//junoUser := junoUsers[0]

	// test IBC conformance before chain upgrade
	//conformance.TestChainPair(t, ctx, client, network, cheqd, juno, rf, rep, r, path)

	// ===================================
	// cheqd user create resources
	// ===================================
	CreateAndUploadDid(t, ctx, "did_payload.json", "resource_payload.json", "revocationList", cheqd, cheqdUser, "5rjaLzcffhGUH4nt4fyfAg", "9fbb1b86-91f8-4942-97b9-725b7714131c")

	//height, err := cheqd.Height(ctx)
	//require.NoError(t, err, "error fetching height before submit upgrade proposal")

	//haltHeight := height + haltHeightDelta

	//urlMap := make(plan.BinaryDownloadURLMap)
	//// we use a small size file, the node checks for the checksum after fetching only
	//urlMap["darwin/amd64"] = "https://github.com/cheqd/did-resolver/releases/download/v3.5.1/did-resolver-3.5.1-darwin-amd64.tar.gz?checksum=sha256:3408e2a65fd1ccb57eee19ff8f42d6c50cc4876fbe91537864be2a335a9fa9dd"

	//upgradeInfo := plan.Info{
	//	Binaries: urlMap,
	//}

	//info, err := json.Marshal(upgradeInfo)
	//require.NoError(t, err, "Marshal info failed")

	//proposal := cosmos.SoftwareUpgradeProposal{
	//	Deposit:     "500000000" + cheqd.Config().Denom, // greater than min deposit
	//	Title:       "Chain Upgrade 1",
	//	Name:        upgradeName,
	//	Description: "First chain software upgrade",
	//	Height:      haltHeight,
	//	Info:        string(info),
	//}

	//upgradeTx, err := cheqd.UpgradeProposal(ctx, cheqdUser.KeyName(), proposal)
	//require.NoError(t, err, "error submitting software upgrade proposal tx")

	//err = cheqd.VoteOnProposalAllValidators(ctx, upgradeTx.ProposalID, cosmos.ProposalVoteYes)
	//require.NoError(t, err, "failed to submit votes")

	//_, err = cosmos.PollForProposalStatus(ctx, cheqd, height, height+haltHeightDelta, upgradeTx.ProposalID, cosmos.ProposalStatusPassed)
	//require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	//height, err = cheqd.Height(ctx)
	//require.NoError(t, err, "error fetching height before upgrade")

	//timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	//defer timeoutCtxCancel()

	//// this should timeout due to chain halt at upgrade height.
	//_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, cheqd)

	//height, err = cheqd.Height(ctx)
	//require.NoError(t, err, "error fetching height after chain should have halted")

	//// make sure that chain is halted
	//require.Equal(t, haltHeight, height, "height is not equal to halt height")

	//// bring down nodes to prepare for upgrade
	//err = cheqd.StopAllNodes(ctx)
	//require.NoError(t, err, "error stopping node(s)")

	//// upgrade version on all nodes
	//cheqd.UpgradeVersion(ctx, client, upgradeVersion)

	//// start all nodes back up.
	//// validators reach consensus on first block after upgrade height
	//// and chain block production resumes.
	//err = cheqd.StartAllNodes(ctx)
	//require.NoError(t, err, "error starting upgraded node(s)")

	//timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	//defer timeoutCtxCancel()

	//err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), cheqd)
	//require.NoError(t, err, "chain did not produce blocks after upgrade")

}
