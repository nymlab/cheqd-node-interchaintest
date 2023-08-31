package cheqd_interchaintest

import (
	"context"
	"testing"

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
	validatorsCount := 1

	ic, chain := CreateCheqdChain(t, ctx, validatorsCount, 0, "sha-5c98ec329797eb7fae0bc40e4b3090b3114e6c24")
	require.NotNil(t, ic)
	require.NotNil(t, chain)

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
	validatorsCount := 1

	ic, chain := CreateCheqdChain(t, ctx, validatorsCount, 0, "v2.0.0-rc1")
	require.NotNil(t, ic)
	require.NotNil(t, chain)

	t.Cleanup(func() {
		_ = ic.Close()
	})
}
