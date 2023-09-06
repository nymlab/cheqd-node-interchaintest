package cheqd_interchaintest

import (
	"context"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v6"
	"github.com/stretchr/testify/require"
)

func TestBasicCheqdV1Start(t *testing.T) {

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
	ic, cheqd, _, _ := CreateCheqdChain(t, ctx, validatorsCount, fullnodeCount, "sha-5c98ec329797eb7fae0bc40e4b3090b3114e6c24")
	require.NotNil(t, ic)
	require.NotNil(t, cheqd)

	const userFunds = int64(10_000_000_000_000)
	cheqdUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, cheqd)
	cheqdUser := cheqdUsers[0]

	CreateAndUploadDid(t, ctx, "did_payload.json", "resource_payload.json", "revocationList", cheqd, cheqdUser, "5rjaLzcffhGUH4nt4fyfAg", "9fbb1b86-91f8-4942-97b9-725b7714131c")

	t.Cleanup(func() {
		_ = ic.Close()
	})

}

func TestBasicCheqdV2Start(t *testing.T) {

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

	ic, cheqd, _, _ := CreateCheqdChain(t, ctx, validatorsCount, fullnodeCount, "sha-5c98ec329797eb7fae0bc40e4b3090b3114e6c24")
	require.NotNil(t, ic)
	require.NotNil(t, cheqd)

	const userFunds = int64(10_000_000_000_000)
	cheqdUsers := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, cheqd)
	cheqdUser := cheqdUsers[0]

	CreateAndUploadDid(t, ctx, "did_payload.json", "resource_payload.json", "revocationList", cheqd, cheqdUser, "5rjaLzcffhGUH4nt4fyfAg", "9fbb1b86-91f8-4942-97b9-725b7714131c")

	t.Cleanup(func() {
		cancelFn()
	})
}
