package cheqd_interchaintest

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7"
	//"github.com/strangelove-ventures/interchaintest/v7/ibc"
	//"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	//"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/require"
)

func TestSSIContract(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancelFn()
	})

	// create a single chain instance with 4 validators
	validatorsCount := 1
	_, chain := CreateChain(t, ctx, validatorsCount, 0)

	// Users
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", int64(10_000_000), chain, chain)
	deployer := users[0]

	// Upload contract
	codeId, err := chain.StoreContract(ctx, deployer.KeyName(), "contracts/vectis_anoncreds_verifier.wasm")
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate contract
	contractAddr, err := chain.InstantiateContract(ctx, deployer.KeyName(), codeId, "{}", true)
	if err != nil {
		t.Fatal(err)
	}

	require.NotNil(t, codeId)
	require.NotNil(t, contractAddr)

}
