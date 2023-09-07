package cheqd_interchaintest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"unicode"

	interchaintest "github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
	"github.com/strangelove-ventures/interchaintest/v6/relayer"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/strangelove-ventures/interchaintest/v6/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

var (
	VotingPeriod     = "15s"
	MaxDepositPeriod = "10s"
)

func TestCheqdV2VectisIBC(t *testing.T) {
	var (
		chainName    = "cheqd"
		cheqdVersion = "v2.0.0-rc2"
	)

	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:        chainName,
			ChainName:   chainName,
			Version:     cheqdVersion,
			ChainConfig: GetCheqdConfig(cheqdVersion),
		},
		{
			Name:        "juno",
			ChainName:   "juno",
			Version:     "v14.1.0",
			ChainConfig: GetJunoConfig()}})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	client, network := interchaintest.DockerSetup(t)
	cheqd, juno := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	const (
		ssiPath     = "ssi-cheqd-juno-path"
		path        = "cheqd-juno-path"
		relayerName = "relayer"
	)

	// Get a relayer instance
	rf := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.StartupFlags("-b", "100"),
	)

	r := rf.Build(t, client, network)

	ic := interchaintest.NewInterchain().
		AddChain(cheqd).
		AddChain(juno).
		AddRelayer(r, relayerName).
		// The default is 'transfer' port on both end with unordered channel
		AddLink(interchaintest.InterchainLink{
			Chain1:  cheqd,
			Chain2:  juno,
			Relayer: r,
			Path:    path,
		})

	ctx := context.Background()

	rep := testreporter.NewNopReporter()

	require.NoError(t, ic.Build(ctx, rep.RelayerExecReporter(t), interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))

	t.Cleanup(func() {
		_ = ic.Close()
	})

	const userFunds = int64(10_000_000_000_000)
	junoUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, juno)
	junoUser := junoUsers[0]
	junoNode := juno.FullNodes[0]

	// ===================================
	// juno user upload and instantiate anoncreds contract
	// ===================================
	codeId, err := juno.StoreContract(ctx, junoUser.KeyName(), "contracts/vectis_anoncreds_verifier.wasm")
	require.NoError(t, err, "code store err")

	_, err = junoNode.ExecTx(ctx, junoUser.KeyName(), "wasm", "instantiate", codeId, "{}", "--label", "vectis-ssi", "--gas", "2000000", "--no-admin")
	require.NoError(t, err, "instantiate err")

	stdout, _, err := junoNode.ExecQuery(ctx, "wasm", "list-contract-by-code", codeId)
	require.NoError(t, err, "Query err")

	contractsRes := QueryContractsByCodeResponse{}
	err = json.Unmarshal([]byte(stdout), &contractsRes)
	require.NoError(t, err, "parse contractRes err")

	contractAddr := contractsRes.Contracts[len(contractsRes.Contracts)-1]

	// ===================================
	// Add channel and make relayer relay it
	// ===================================

	createChannelOptions := ibc.CreateChannelOptions{
		SourcePortName: "cheqdresource",
		DestPortName:   fmt.Sprintf("wasm.%s", contractAddr),
		Order:          ibc.Unordered,
		Version:        "cheqd-resource-v3",
	}

	err = r.GeneratePath(ctx, rep.RelayerExecReporter(t), "cheqd-mainnet-1", "juno-mainnet-1", ssiPath)
	require.NoError(t, err, "generate path relayer err")
	err = r.LinkPath(ctx, rep.RelayerExecReporter(t), ssiPath, createChannelOptions, ibc.DefaultClientOpts())
	// These do not actually return error if they do not succeed in making the channel
	require.NoError(t, err, "create channel relayer err")

	// Ensure channels are created successfully
	channelsCheqd, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), "cheqd-mainnet-1")
	require.Len(t, channelsCheqd, 2)

	// ===================================
	// cheqd user create resource
	// ===================================
	cheqdUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, cheqd)
	cheqdUser := cheqdUsers[0]
	CreateAndUploadDid(t, ctx, "did_payload.json", "resource_payload.json", "revocationList", cheqd, cheqdUser, "5rjaLzcffhGUH4nt4fyfAg", "9fbb1b86-91f8-4942-97b9-725b7714131c")

	// ============================================================
	// Upload vectis-ssi contract on remote cosmwasm enabled chain
	// ============================================================
	err = juno.ExecuteContract(ctx, junoUser.KeyName(), contractAddr, fmt.Sprintf(`{"update_state": {"resource_id": "%s", "collection_id": "%s" }}`, TestResourceId, TestCollectionId))
	require.NoError(t, err, "exec error err")

	height, err := juno.Height(ctx)
	require.NoError(t, err, "error fetching height before flush")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*12)
	defer timeoutCtxCancel()
	_ = testutil.WaitForBlocks(timeoutCtx, int(height)+3, juno)

	for _, channel := range channelsCheqd {
		// we do not check if flushing has error because channels can be for different paths
		r.FlushPackets(ctx, rep.RelayerExecReporter(t), ssiPath, channel.ChannelID)
		r.FlushAcknowledgements(ctx, rep.RelayerExecReporter(t), ssiPath, channel.ChannelID)
	}

	var queryData QueryResultResourceWithMetadata
	query, err := json.Marshal(QueryMsg{QueryState: &QueryStateInput{ResourceId: TestResourceId, CollectionId: TestCollectionId}})
	require.NoError(t, err, "query parse err")

	stdout, _, err = junoNode.ExecQuery(ctx, "wasm", "contract-state", "smart", contractAddr, string(query))
	require.NoError(t, err, "exec err")

	err = json.Unmarshal(stdout, &queryData)
	require.NoError(t, err, "ack parse err")

	resourceFromContract := strings.TrimFunc(string(queryData.Data.GetResource().GetData()), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	content, err := os.ReadFile(fmt.Sprintf("%s/%s", "artifacts", "revocationList"))
	originalResource := strings.TrimFunc(string(content), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	require.Equal(t, resourceFromContract, originalResource)

}
