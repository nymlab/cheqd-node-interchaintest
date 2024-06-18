package cheqd_interchaintest

import (
	"context"
	"runtime"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/stretchr/testify/require"
)

func TestCheqdV2UploadResource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	ctx, cancelFn := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancelFn()
	})

	// create a single chain instance with x validators
	validatorsCount, fullnodeCount := 1, 1

	var cheqd_version string
	var repository string

	if runtime.GOARCH == "arm64" {
		cheqd_version = "v2.0.1-arm64"
		repository = "ghcr.io/nymlab/cheqd-node"
	} else {
		cheqd_version = "sha-fdf3b2cb9bef2ee518f46e299eee97b4c4082ff2"
		repository = "ghcr.io/cheqd/cheqd-node"
	}

	ic, cheqd, _, _ := CreateCheqdChain(
		t,
		ctx,
		validatorsCount,
		fullnodeCount,
		cheqd_version,
		repository,
	)
	require.NotNil(t, ic)
	require.NotNil(t, cheqd)

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

	t.Cleanup(func() {
		cancelFn()
	})
}
