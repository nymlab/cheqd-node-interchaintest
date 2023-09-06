package cheqd_interchaintest

import (
	resourcetypes "github.com/cheqd/cheqd-node/x/resource/types"
)

// from cosmos-sdk v0.45.15
type QueryContractsByCodeResponse struct {
	// contracts are a set of contract addresses
	Contracts []string `protobuf:"bytes,1,rep,name=contracts,proto3" json:"contracts,omitempty"`
	// pagination defines the pagination in the response.
	Pagination *PageResponse `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

type PageResponse struct {
	// next_key is the key to be passed to PageRequest.key to
	// query the next page most efficiently
	NextKey []byte `protobuf:"bytes,1,opt,name=next_key,json=nextKey,proto3" json:"next_key,omitempty"`
	// total is total number of results available if PageRequest.count_total
	// was set, its value is undefined otherwise
	Total string `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
}

type QueryMsg struct {
	// Tokenfactory Core
	QueryState *QueryStateInput `json:"query_state,omitempty"`
}

type QueryStateInput struct {
	ResourceId   string `json:"resource_id"`
	CollectionId string `json:"collection_id"`
}

type QueryResultResourceWithMetadata struct {
	Data resourcetypes.ResourceWithMetadata `json:"data"`
}
