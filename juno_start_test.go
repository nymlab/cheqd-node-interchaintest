package cheqd_interchaintest

import (
	"context"
	"fmt"
	"testing"

	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/stretchr/testify/require"
)

func TestJunoStart(t *testing.T) {

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

	ic, juno, _, _ := CreateJunoChain(
		t,
		ctx,
		validatorsCount,
		fullnodeCount,
	)
	require.NotNil(t, ic)
	require.NotNil(t, juno)

	const userFunds = int64(10_000_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, juno)
	user := users[0]

	fmt.Println("user: ", user)

	//CreateAndUploadDid(
	//	t,
	//	ctx,
	//	"did_payload.json",
	//	"resource_payload.json",
	//	"revocationList",
	//	cheqd,
	//	cheqdUser,
	//	"5rjaLzcffhGUH4nt4fyfAg",
	//	"9fbb1b86-91f8-4942-97b9-725b7714131c",
	//)

	t.Cleanup(func() {
		cancelFn()
	})
}
