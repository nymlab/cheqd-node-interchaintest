package cheqd_interchaintest

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	cheqdCli "github.com/cheqd/cheqd-node/x/did/client/cli"
	resourcetypes "github.com/cheqd/cheqd-node/x/resource/types"
)

func TestJsonparse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	payloadJSON, _, err := cheqdCli.ReadPayloadWithSignInputsFromFile(
		"artifacts/resource_payload_no_data.json",
	)

	// Unmarshal payload
	var payload resourcetypes.MsgCreateResourcePayload
	err = json.Unmarshal(payloadJSON, &payload)

	require.NoError(t, err, "parse")
}
