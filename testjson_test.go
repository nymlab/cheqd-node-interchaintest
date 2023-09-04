package cheqd_interchaintest

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	cheqdCli "github.com/cheqd/cheqd-node/x/did/client/cli"
	resourcetypes "github.com/cheqd/cheqd-node/x/resource/types"
)

func TestJsonparse(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	payloadJSON, _, err := cheqdCli.ReadPayloadWithSignInputsFromFile("artifacts/resource_payload.json")

	// Unmarshal payload
	var payload resourcetypes.MsgCreateResourcePayload
	err = json.Unmarshal(payloadJSON, &payload)

	fmt.Println(payload)
	require.NoError(t, err, "aprse")
}
