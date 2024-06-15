package cheqd_interchaintest

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"testing"

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
	t.Parallel()

	var cheqd_version string
	var repository string

	if runtime.GOARCH == "arm64" {
		cheqd_version = "v2.0.1-arm64"
		repository = "ghcr.io/nymlab/cheqd-node"
	} else {
		cheqd_version = "sha-fdf3b2cb9bef2ee518f46e299eee97b4c4082ff2"
		repository = "ghcr.io/cheqd/cheqd-node"
	}

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:        "cheqd",
			ChainName:   "cheqd",
			Version:     cheqd_version,
			ChainConfig: GetCheqdConfig(cheqd_version, repository),
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

	junoUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, juno)
	junoUser := junoUsers[0]
	junoNode := juno.FullNodes[0]

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
		collectionId,
		resourceId,
	)

	// ===================================
	// juno user upload and instantiate sdjwt contract
	// ===================================
	codeId, err := juno.StoreContract(
		ctx,
		junoUser.KeyName(),
		contractPath,
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

	// ======================================
	// Add channel and make relayer relay it
	// ======================================

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

	err = testutil.WaitForBlocks(ctx, 5, juno, cheqd)
	require.NoError(t, err, "wait for blocks err")

	// ============================================================
	// Register route on contract with cheqd as trust registry
	// ============================================================
	ResourceReq := resourcetypes.ResourceReqPacket{
		CollectionId: collectionId,
		ResourceId:   resourceId,
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

	err = testutil.WaitForBlocks(ctx, 5, juno, cheqd)
	require.NoError(t, err, "wait for blocks err")

	for _, channel := range channelsCheqd {
		r.Flush(ctx, rep.RelayerExecReporter(t), ssiPath, channel.ChannelID)
	}

	err = testutil.WaitForBlocks(ctx, 10, juno, cheqd)
	require.NoError(t, err, "wait for blocks err")

	for _, channel := range channelsCheqd {
		r.Flush(ctx, rep.RelayerExecReporter(t), ssiPath, channel.ChannelID)
	}

	// =================================================
	// Query contract for verification key
	// This is retrieved from IBC tx on cheqe x/resource
	// =================================================
	query, err := json.Marshal(sdjwttypes.QueryMsg{
		GetRouteVerificationKey: &sdjwttypes.GetRouteVerificationKey{
			AppAddr: sdjwttypes.TestAppAddr2,
			RouteID: 1,
		},
	})

	var queryData sdjwttypes.GetRouteVerificationKeyRes
	err = junoNode.QueryContract(ctx, contractAddr, string(query), &queryData)

	var originalJwk sdjwttypes.OkpJwk
	var returnedJwk sdjwttypes.OkpJwk

	err = json.Unmarshal(jwk, &originalJwk)
	err = json.Unmarshal([]byte(queryData.Data), &returnedJwk)

	require.Equal(t, originalJwk, returnedJwk)
}
