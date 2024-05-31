package cheqd_interchaintest

import (
	resourcetypes "github.com/cheqd/cheqd-node/x/resource/types"
)

type QueryMsg struct {
	QueryState *QueryStateInput `json:"query_state,omitempty"`
}

type QueryStateInput struct {
	ResourceId   string `json:"resource_id"`
	CollectionId string `json:"collection_id"`
}

type QueryResultResourceWithMetadata struct {
	Data resourcetypes.ResourceWithMetadata `json:"data"`
}
