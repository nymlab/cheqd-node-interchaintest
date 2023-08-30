package cheqd_interchaintest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicJunoStart(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, cancelFn := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancelFn()
	})

	// create a single chain instance with x validators
	validatorsCount := 1

	ic, chain := CreateChain(t, ctx, validatorsCount, 0)
	require.NotNil(t, ic)
	require.NotNil(t, chain)

	t.Cleanup(func() {
		_ = ic.Close()
	})
}
