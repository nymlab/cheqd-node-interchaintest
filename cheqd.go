package cheqd_interchaintest

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
)

func MustUploadPayload(
	ctx context.Context,
	filepath string,
	filename string,
	chain *cosmos.CosmosChain,
) error {
	content, err := os.ReadFile(fmt.Sprintf("%s/%s", filepath, filename))
	str := fmt.Sprintf(
		`touch /var/cosmos-chain/cheqd/%s && echo '%s' > /var/cosmos-chain/cheqd/%s`,
		filename,
		string(content),
		filename,
	)
	_, _, err = chain.Exec(ctx, []string{"sh", "-c", str}, []string{})
	return err
}

func CreateAndUploadDid(
	t *testing.T,
	ctx context.Context,
	didPayload string,
	resourcePayload string,
	resourceFile string,
	chain *cosmos.CosmosChain,
	user ibc.Wallet,
	collectionId string,
	resourceId string,
) []byte {

	err := MustUploadPayload(ctx, "artifacts", didPayload, chain)
	require.NoError(t, err, "upload file err")
	err = MustUploadPayload(ctx, "artifacts", resourcePayload, chain)
	require.NoError(t, err, "upload file err")
	err = MustUploadPayload(ctx, "artifacts", resourceFile, chain)

	nodes := chain.Nodes()
	chainNode := nodes[len(nodes)-1]

	require.NoError(t, err, "upload file err")
	_, err = chainNode.ExecTx(
		ctx,
		user.KeyName(),
		"cheqd",
		"create-did",
		fmt.Sprintf("/var/cosmos-chain/cheqd/%s", didPayload),
		"--gas",
		"200000",
		"--fees",
		"50000000000ncheq",
	)
	require.NoError(t, err, "create-did err")

	_, err = chainNode.ExecTx(
		ctx,
		user.KeyName(),
		"resource",
		"create",
		fmt.Sprintf("/var/cosmos-chain/cheqd/%s", resourcePayload),
		fmt.Sprintf("/var/cosmos-chain/cheqd/%s", resourceFile),
		"--gas",
		"200000",
		"--fees",
		"50000000000ncheq",
	)
	require.NoError(t, err, "create-resource err")

	// Query onchain data
	res1, _, err := chainNode.ExecQuery(ctx, "resource", "collection-metadata", collectionId)
	require.NoError(t, err, "query-collection err")
	require.NotNil(t, res1, "return collection err")

	res1, _, err = chainNode.ExecQuery(
		ctx,
		"resource",
		"specific-resource",
		collectionId,
		resourceId,
	)
	require.NoError(t, err, "query-resource err")
	require.NotNil(t, res1, "return resource err")
	return (res1)
}
