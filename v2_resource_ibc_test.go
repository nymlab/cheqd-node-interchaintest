package cheqd_interchaintest

import (
	"context"
	"encoding/json"
	"fmt"
	//"os"
	//"strings"
	"testing"
	"time"
	//"unicode"

	sdjwttypes "github.com/nymlab/cheqd-interchaintest/types"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/relayer"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	resourcetypes "github.com/cheqd/cheqd-node/x/resource/types"
)

func TestCheqdV2AvidaIbc(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:        "cheqd",
			ChainName:   "cheqd",
			Version:     "v2.0.1-arm64",
			ChainConfig: GetCheqdConfig(),
			NoHostMount: &[]bool{false}[0], // specify no mount
		},
		{
			Name:        "juno",
			ChainName:   "juno",
			Version:     "v22.0.0",
			ChainConfig: GetJunoConfig(),
			NoHostMount: &[]bool{false}[0], // specify no mount
		}})

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

	require.NoError(
		t,
		ic.Build(ctx, rep.RelayerExecReporter(t), interchaintest.InterchainBuildOptions{
			TestName:         t.Name(),
			Client:           client,
			NetworkID:        network,
			SkipPathCreation: false,
		}),
	)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	const userFunds = int64(10_000_000_000_000)
	junoUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, juno)
	junoUser := junoUsers[0]
	junoNode := juno.FullNodes[0]

	// ===================================
	// juno user upload and instantiate sdjwt contract
	// ===================================
	codeId, err := juno.StoreContract(
		ctx,
		junoUser.KeyName(),
		"contracts/avida_sdjwt_verifier-aarch64.wasm",
	)
	require.NoError(t, err, "code store err")

	routeReqs := make([]sdjwttypes.RouteRequirement, 0)

	initRegistrations := make([]sdjwttypes.InitRegistration, 0)
	var initMsg sdjwttypes.InstantiateMsg

	initMsg = sdjwttypes.InstantiateMsg{
		InitRegistrations: append(initRegistrations, sdjwttypes.InitRegistration{
			AppAdmin:   sdjwttypes.TestAppAddr1,
			AppAddress: sdjwttypes.TestAppAddr1,
			Routes:     routeReqs,
		}),
		MaxPresentationLen: 30000,
	}

	initMsgBytes, err := json.Marshal(initMsg)

	contractAddr, err := junoNode.InstantiateContract(
		ctx,
		junoUser.KeyName(),
		codeId,
		string(initMsgBytes),
		true,
		"--label",
		"avida-sdjwt",
		"--gas",
		"2000000",
	)
	require.NoError(t, err, "instantiate err")

	// ===================================
	// Add channel and make relayer relay it
	// ===================================

	createChannelOptions := ibc.CreateChannelOptions{
		SourcePortName: "cheqdresource",
		DestPortName:   fmt.Sprintf("wasm.%s", contractAddr),
		Order:          ibc.Unordered,
		Version:        "cheqd-resource-v3",
	}

	err = r.GeneratePath(
		ctx,
		rep.RelayerExecReporter(t),
		"cheqd-mainnet-1",
		"juno-mainnet-1",
		ssiPath,
	)
	require.NoError(t, err, "generate path relayer err")
	err = r.LinkPath(
		ctx,
		rep.RelayerExecReporter(t),
		ssiPath,
		createChannelOptions,
		ibc.DefaultClientOpts(),
	)
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
	CreateAndUploadDid(
		t,
		ctx,
		"did_payload.json",
		"resource_payload_no_data.json",
		"jwk.json",
		cheqd,
		cheqdUser,
		"5rjaLzcffhGUH4nt4fyfAg",
		"9fbb1b86-91f8-4942-97b9-725b7714131c",
	)

	// ============================================================
	// Register route on contract with cheqd as trust registry
	// ============================================================
	ResourceReq := resourcetypes.ResourceReqPacket{
		CollectionId: "5rjaLzcffhGUH4nt4fyfAg",
		ResourceId:   "9fbb1b86-91f8-4942-97b9-725b7714131c",
	}
	resourceReqBytes, err := json.Marshal(ResourceReq)

	routeReq := sdjwttypes.RouteRequirement{
		RouteId: 1,
		Requirements: sdjwttypes.RouteVerificationRequirements{
			PresentationRequest: []byte("[]"),
			VerificationSource: sdjwttypes.VerificationSource{
				DataOrLocation: resourceReqBytes,
				Source:         sdjwttypes.TrustRegistryCheqd,
			},
		},
	}
	registerMsg := sdjwttypes.ExecuteMsg{
		Register: &sdjwttypes.Register{
			AppAddr:       sdjwttypes.TestAppAddr2,
			RouteCriteria: append(routeReqs, routeReq)},
	}

	registerMsgBytes, err := json.Marshal(registerMsg)

	_, err = juno.ExecuteContract(
		ctx,
		junoUser.KeyName(),
		contractAddr,
		string(registerMsgBytes),
	)
	require.NoError(t, err, "exec err")

	height, err := juno.Height(ctx)
	require.NoError(t, err, "error fetching height before flush")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*12)
	defer timeoutCtxCancel()
	_ = testutil.WaitForBlocks(timeoutCtx, int(height)+3, juno)

	for _, channel := range channelsCheqd {
		// we do not check if flushing has error because channels can be for different paths
		r.Flush(ctx, rep.RelayerExecReporter(t), ssiPath, channel.ChannelID)
	}

	// ===========================================
	// Query contract for verification key on cheqd
	// ===========================================

	query, err := json.Marshal(sdjwttypes.QueryMsg{
		GetRouteVerificationKey: &sdjwttypes.GetRouteVerificationKey{
			AppAddr: sdjwttypes.TestAppAddr2,
			RouteID: 1,
		},
	})

	var queryData sdjwttypes.GetRouteVerificationKeyRes
	err = junoNode.QueryContract(ctx, contractAddr, string(query), &queryData)

	fmt.Println("queryData: ", queryData)

	//resourceFromContract := strings.TrimFunc(
	//	string(queryData.Data.GetResource().GetData()),
	//	func(r rune) bool {
	//		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	//	},
	//)

	//content, err := os.ReadFile(fmt.Sprintf("%s/%s", "artifacts", "revocationList"))
	//originalResource := strings.TrimFunc(string(content), func(r rune) bool {
	//	return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	//})

	//require.Equal(t, resourceFromContract, originalResource)

}
